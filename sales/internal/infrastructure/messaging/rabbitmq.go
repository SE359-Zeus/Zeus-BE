package messaging

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	OrderCreatedQueue    = "sales.order.created"
	OrderAllocatedQueue  = "sales.order.allocated"
	OrderCancelledQueue  = "sales.order.cancelled"
	ClientUpdatedQueue   = "sales.client.updated"
	AuditQueue           = "system.audit.log"
	FulfillmentQueued    = "sales.fulfillment.queued"
	FulfillmentProcessed = "sales.fulfillment.processed"
)

var ErrUnavailable = errors.New("rabbitmq unavailable")

type Publisher interface {
	Publish(ctx context.Context, queue string, payload any) error
}

type RabbitMQ struct {
	url string
}

func NewRabbitMQ(url string) (*RabbitMQ, error) {
	if url == "" {
		slog.Info("rabbitmq disabled", slog.String("service", "sales"), slog.String("component", "rabbitmq"), slog.String("reason", "no_url_configured"))
		return nil, ErrUnavailable
	}
	slog.Info("rabbitmq client initialized", slog.String("service", "sales"), slog.String("component", "rabbitmq"))
	return &RabbitMQ{url: url}, nil
}

func (r *RabbitMQ) Ping(ctx context.Context) error {
	if r == nil || r.url == "" {
		return ErrUnavailable
	}
	conn, channel, err := dialChannel(r.url)
	if err != nil {
		slog.Warn("rabbitmq connection failed", slog.String("service", "sales"), slog.String("component", "rabbitmq"), slog.String("url", r.url), slog.String("error", err.Error()))
		return err
	}
	defer conn.Close()
	defer channel.Close()
	if err := declareQueues(channel); err != nil {
		slog.Warn("rabbitmq queue declaration failed", slog.String("service", "sales"), slog.String("component", "rabbitmq"), slog.String("url", r.url), slog.String("error", err.Error()))
		return err
	}
	slog.Info("rabbitmq connection successful", slog.String("service", "sales"), slog.String("component", "rabbitmq"), slog.String("url", r.url))
	return nil
}

func (r *RabbitMQ) Publish(ctx context.Context, queue string, payload any) error {
	return r.withChannel(func(channel *amqp.Channel) error {
		if err := declareQueues(channel); err != nil {
			return err
		}
		body, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		return channel.PublishWithContext(ctx, "", queue, true, false, amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent,
			Timestamp:    time.Now().UTC(),
		})
	})
}

func declareQueues(channel *amqp.Channel) error {
	for _, queue := range []string{
		OrderCreatedQueue,
		OrderAllocatedQueue,
		OrderCancelledQueue,
		ClientUpdatedQueue,
		AuditQueue,
		FulfillmentQueued,
		FulfillmentProcessed,
	} {
		if _, err := channel.QueueDeclare(queue, true, false, false, false, nil); err != nil {
			return err
		}
	}
	return nil
}

func (r *RabbitMQ) withChannel(fn func(*amqp.Channel) error) error {
	if r == nil || r.url == "" {
		return ErrUnavailable
	}
	conn, channel, err := dialChannel(r.url)
	if err != nil {
		return err
	}
	defer conn.Close()
	defer channel.Close()
	return fn(channel)
}

func dialChannel(url string) (*amqp.Connection, *amqp.Channel, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}
	channel, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, nil, fmt.Errorf("failed to open channel: %w", err)
	}
	return conn, channel, nil
}
