package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"zeus-mrp-service/internal/infrastructure/messaging"
	"zeus-mrp-service/internal/infrastructure/observability"
	"zeus-mrp-service/internal/models"
	"zeus-mrp-service/internal/service"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
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

	if err := messaging.DeclareQueues(conn, ch); err != nil {
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

	// Consume deficit reserved events
	reservedMsgs, err := ch.Consume(messaging.DeficitReservedQueue, "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("failed to consume from %s: %w", messaging.DeficitReservedQueue, err)
	}

	slog.Info("MRP sales order consumer started", slog.String("service", "mrp"))

	go c.loop(ctx, createdMsgs, "created")
	go c.loop(ctx, updatedMsgs, "updated")
	go c.loopReserved(ctx, reservedMsgs)

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

			// Extract trace context from AMQP headers using OTel propagator
			headers := propagation.MapCarrier{}
			traceID := ""
			spanID := ""
			if msg.Headers != nil {
				// Helper to get header value in case-insensitive way
				getHeader := func(key string) any {
					lowerKey := strings.ToLower(key)
					for k, v := range msg.Headers {
						if strings.ToLower(k) == lowerKey {
							return v
						}
					}
					return nil
				}

				// Try traceparent first
				if tpVal := getHeader("traceparent"); tpVal != nil {
					var tpStr string
					switch v := tpVal.(type) {
					case string:
						tpStr = v
					case []byte:
						tpStr = string(v)
					}
					if tpStr != "" {
						headers.Set("traceparent", tpStr)
						parts := strings.Split(tpStr, "-")
						if len(parts) == 4 {
							if len(parts[1]) == 32 {
								traceID = parts[1]
							}
							if len(parts[2]) == 16 {
								spanID = parts[2]
							}
						}
					}
				}

				// Fallback to trace_id and span_id
				if traceID == "" {
					if tidVal := getHeader("trace_id"); tidVal != nil {
						switch v := tidVal.(type) {
						case string:
							traceID = v
						case []byte:
							traceID = string(v)
						}
					}
				}
				if spanID == "" {
					if sidVal := getHeader("span_id"); sidVal != nil {
						switch v := sidVal.(type) {
						case string:
							spanID = v
						case []byte:
							spanID = string(v)
						}
					}
				}

				// Ensure traceparent is set in map carrier if traceID is available
				if traceID != "" && headers.Get("traceparent") == "" {
					if spanID == "" {
						spanID = observability.NewSpanID()
					}
					headers.Set("traceparent", fmt.Sprintf("00-%s-%s-01", traceID, spanID))
				}
			}
			propagator := otel.GetTextMapPropagator()
			msgCtx := propagator.Extract(ctx, headers)

			// Start an OTel child span so it gets exported to the tracing backend
			tracer := otel.Tracer("mrp")
			msgCtx, span := tracer.Start(msgCtx, fmt.Sprintf("consume sales.order.%s", eventType))

			sc := span.SpanContext()
			if sc.IsValid() && sc.TraceID().String() != "00000000000000000000000000000000" {
				traceID = sc.TraceID().String()
				spanID = sc.SpanID().String()
			} else {
				if traceID == "" {
					traceID = observability.NewTraceID()
				}
				if spanID == "" {
					spanID = observability.NewSpanID()
				}
			}

			msgCtx = observability.WithTraceID(msgCtx, traceID)
			msgCtx = observability.WithSpanID(msgCtx, spanID)

			slog.InfoContext(msgCtx, "received sales order event",
				slog.String("service", "mrp"),
				slog.String("event_type", eventType),
				slog.String("trace_id", traceID),
				slog.String("extracted_span_id", spanID),
				slog.Bool("otel_span_valid", sc.IsValid()),
				slog.String("otel_trace_id", sc.TraceID().String()),
				slog.String("otel_span_id", sc.SpanID().String()),
				slog.Any("msg_headers", msg.Headers),
				slog.String("body", string(msg.Body)),
			)

			var payload OrderCreatedPayload
			if err := json.Unmarshal(msg.Body, &payload); err != nil {
				slog.ErrorContext(msgCtx, "failed to unmarshal sales order event payload", slog.String("service", "mrp"), slog.String("error", err.Error()))
				span.End()
				_ = msg.Nack(false, false) // discard invalid format messages
				continue
			}

			if payload.OrderID == "" {
				slog.WarnContext(msgCtx, "received sales order event with empty order_id, discarding", slog.String("service", "mrp"))
				span.End()
				_ = msg.Nack(false, false)
				continue
			}

			// Process demand creation logic from queue payload only.
			if err := c.processOrderPayload(msgCtx, payload); err != nil {
				slog.ErrorContext(msgCtx, "failed to process sales order event", slog.String("service", "mrp"), slog.String("order_id", payload.OrderID), slog.String("error", err.Error()))
				span.End()
				_ = msg.Nack(false, true) // Requeue for transient errors
			} else {
				slog.InfoContext(msgCtx, "successfully processed sales order event", slog.String("service", "mrp"), slog.String("order_id", payload.OrderID))
				span.End()
				_ = msg.Ack(false)
			}
		}
	}
}

func (c *OrderConsumer) processOrderPayload(ctx context.Context, payload OrderCreatedPayload) error {
	if len(payload.Items) == 0 {
		slog.WarnContext(ctx, "sales order event has no items, skipping production planning",
			slog.String("service", "mrp"),
			slog.String("order_id", payload.OrderID),
		)
		return nil
	}

	for _, item := range payload.Items {
		requestedQty := item.requestedQuantity()
		if item.SKU == "" || requestedQty <= 0 {
			slog.WarnContext(ctx, "sales order item is invalid, skipping item",
				slog.String("service", "mrp"),
				slog.String("order_id", payload.OrderID),
				slog.String("sku", item.SKU),
				slog.Int("qty", requestedQty),
			)
			continue
		}

		slog.InfoContext(ctx, "triggering production planning for sales order item",
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

func (c *OrderConsumer) loopReserved(ctx context.Context, msgs <-chan amqp.Delivery) {
	for {
		select {
		case <-ctx.Done():
			slog.Info("MRP deficit reserved consumer loop stopping due to context cancellation")
			return
		case msg, ok := <-msgs:
			if !ok {
				slog.Info("MRP deficit reserved consumer message channel closed")
				return
			}

			// We just trigger a global invalidation since specific order ID might not be present,
			// or SCM satisfies a component shortage which affects multiple open orders.
			slog.Info("received deficit reserved event, invalidating global readiness cache")
			c.mrpService.InvalidateReadinessCache(ctx, uuid.Nil)

			_ = msg.Ack(false)
		}
	}
}
