package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"zeus-mrp-service/internal/infrastructure/messaging"
	"zeus-mrp-service/internal/models"
	"zeus-mrp-service/internal/service"

	amqp "github.com/rabbitmq/amqp091-go"
)

type OrderCreatedPayload struct {
	OrderID      string             `json:"order_id"`
	ClientID     string             `json:"client_id"`
	Total        float64            `json:"total"`
	RequiredDate time.Time          `json:"required_date"`
	Items        []OrderPayloadItem `json:"items"`
}

type OrderPayloadItem struct {
	SKU          string `json:"sku"`
	RequestedQty int    `json:"requested_qty"`
	Qty          int    `json:"qty"`
}

func (i OrderPayloadItem) requestedQuantity() int {
	if i.RequestedQty > 0 {
		return i.RequestedQty
	}
	return i.Qty
}

type OrderConsumer struct {
	mqURL      string
	mrpService *service.ProductionService
	conn       *amqp.Connection
	channel    *amqp.Channel
}

func NewOrderConsumer(mqURL string, mrpService *service.ProductionService) *OrderConsumer {
	return &OrderConsumer{
		mqURL:      mqURL,
		mrpService: mrpService,
	}
}

func (c *OrderConsumer) Start(ctx context.Context) error {
	if c.mqURL == "" {
		slog.Warn("rabbitmq URL is empty, skipping consumer start", slog.String("service", "mrp"))
		return nil
	}

	conn, err := amqp.Dial(c.mqURL)
	if err != nil {
		return fmt.Errorf("failed to connect to rabbitmq: %w", err)
	}
	c.conn = conn

	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("failed to open channel: %w", err)
	}
	c.channel = ch

	if err := messaging.DeclareQueues(ch); err != nil {
		return err
	}

	// Consume created events
	createdMsgs, err := ch.Consume(messaging.SalesOrderCreatedQueue, "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("failed to consume from %s: %w", messaging.SalesOrderCreatedQueue, err)
	}

	// Consume updated events
	updatedMsgs, err := ch.Consume(messaging.SalesOrderUpdatedQueue, "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("failed to consume from %s: %w", messaging.SalesOrderUpdatedQueue, err)
	}

	slog.Info("MRP sales order consumer started", slog.String("service", "mrp"))

	go c.loop(ctx, createdMsgs, "created")
	go c.loop(ctx, updatedMsgs, "updated")

	return nil
}

func (c *OrderConsumer) Close() {
	if c.channel != nil {
		_ = c.channel.Close()
	}
	if c.conn != nil {
		_ = c.conn.Close()
	}
}

func (c *OrderConsumer) loop(ctx context.Context, msgs <-chan amqp.Delivery, eventType string) {
	for {
		select {
		case <-ctx.Done():
			slog.Info("MRP sales order consumer loop stopping due to context cancellation", slog.String("event_type", eventType))
			return
		case msg, ok := <-msgs:
			if !ok {
				slog.Info("MRP sales order consumer message channel closed", slog.String("event_type", eventType))
				return
			}

			slog.Info("received sales order event", slog.String("service", "mrp"), slog.String("event_type", eventType), slog.String("body", string(msg.Body)))

			var payload OrderCreatedPayload
			if err := json.Unmarshal(msg.Body, &payload); err != nil {
				slog.Error("failed to unmarshal sales order event payload", slog.String("service", "mrp"), slog.String("error", err.Error()))
				_ = msg.Nack(false, false) // discard invalid format messages
				continue
			}

			if payload.OrderID == "" {
				slog.Warn("received sales order event with empty order_id, discarding", slog.String("service", "mrp"))
				_ = msg.Nack(false, false)
				continue
			}

			// Process demand creation logic from queue payload only.
			if err := c.processOrderPayload(ctx, payload); err != nil {
				slog.Error("failed to process sales order event", slog.String("service", "mrp"), slog.String("order_id", payload.OrderID), slog.String("error", err.Error()))
				// Requeue for transient errors, e.g. network timeout calling Sales API
				_ = msg.Nack(false, true)
			} else {
				slog.Info("successfully processed sales order event", slog.String("service", "mrp"), slog.String("order_id", payload.OrderID))
				_ = msg.Ack(false)
			}
		}
	}
}

func (c *OrderConsumer) processOrderPayload(ctx context.Context, payload OrderCreatedPayload) error {
	if len(payload.Items) == 0 {
		slog.Warn("sales order event has no items, skipping production planning",
			slog.String("service", "mrp"),
			slog.String("order_id", payload.OrderID),
		)
		return nil
	}

	for _, item := range payload.Items {
		requestedQty := item.requestedQuantity()
		if item.SKU == "" || requestedQty <= 0 {
			slog.Warn("sales order item is invalid, skipping item",
				slog.String("service", "mrp"),
				slog.String("order_id", payload.OrderID),
				slog.String("sku", item.SKU),
				slog.Int("qty", requestedQty),
			)
			continue
		}

		slog.Info("triggering production planning for sales order item",
			slog.String("service", "mrp"),
			slog.String("order_id", payload.OrderID),
			slog.String("sku", item.SKU),
			slog.Int("qty", requestedQty),
		)

		req := models.CreateProductionOrderRequest{
			ProductModelCode: item.SKU,
			TargetQuantity:   requestedQty,
			ScheduledAt:      payload.RequiredDate,
		}

		if _, err := c.mrpService.PlanProduction(ctx, req); err != nil {
			return fmt.Errorf("failed to plan production for SKU %s: %w", item.SKU, err)
		}
	}

	return nil
}
