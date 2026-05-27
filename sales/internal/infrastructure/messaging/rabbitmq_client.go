package messaging

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"
	"zeus-sales-service/internal/infrastructure/observability"

	amqp "github.com/rabbitmq/amqp091-go"
)

var ErrUnavailable = errors.New("rabbitmq unavailable")

type Publisher interface {
	Publish(ctx context.Context, queue string, payload any) error
}

type RabbitMQ struct {
	url       string
	available bool
	mu        sync.RWMutex
}

func NewRabbitMQ(url string) *RabbitMQ {
	r := &RabbitMQ{url: url}
	if url == "" {
		slog.Info("rabbitmq disabled", slog.String("service", "sales"), slog.String("component", "rabbitmq"), slog.String("reason", "no_url_configured"))
		return r
	}

	conn, err := amqp.Dial(url)
	if err != nil {
		slog.Warn("rabbitmq connection failed", slog.String("service", "sales"), slog.String("component", "rabbitmq"), slog.String("url", url), slog.String("error", err.Error()))
		return r
	}
	_ = conn.Close()

	r.available = true
	slog.Info("rabbitmq connection successful", slog.String("service", "sales"), slog.String("component", "rabbitmq"), slog.String("url", url))
	return r
}

func (r *RabbitMQ) isAvailable() bool {
	if r == nil {
		return false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.available && r.url != ""
}

func (r *RabbitMQ) Publish(ctx context.Context, queue string, payload any) error {
	return r.withChannel(func(channel *amqp.Channel) error {
		if err := declareQueues(channel); err != nil {
			slog.Error("rabbitmq queue declaration failed",
				slog.String("service", "sales"),
				slog.String("component", "rabbitmq"),
				slog.String("event", "publish"),
				slog.String("queue", queue),
				slog.String("error", err.Error()),
			)
			return err
		}
		body, err := json.Marshal(payload)
		if err != nil {
			slog.Error("rabbitmq payload marshal failed",
				slog.String("service", "sales"),
				slog.String("component", "rabbitmq"),
				slog.String("event", "publish"),
				slog.String("queue", queue),
				slog.String("error", err.Error()),
			)
			return err
		}
		headers := amqp.Table{}
		traceID := observability.TraceIDFromContext(ctx)
		if traceID != "" {
			spanID := observability.SpanIDFromContext(ctx)
			if spanID == "" {
				spanID = observability.NewSpanID()
			}
			headers["traceparent"] = "00-" + traceID + "-" + spanID + "-01"
			headers["trace_id"] = traceID
			headers["span_id"] = spanID
		}

		err = channel.PublishWithContext(ctx, "", queue, true, false, amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent,
			Timestamp:    time.Now().UTC(),
			Headers:      headers,
		})
		if err != nil {
			slog.ErrorContext(ctx, "rabbitmq publish failed",
				slog.String("service", "sales"),
				slog.String("component", "rabbitmq"),
				slog.String("event", "publish"),
				slog.String("queue", queue),
				slog.Int("payload_bytes", len(body)),
				slog.String("error", err.Error()),
			)
			return err
		}

		slog.InfoContext(ctx, "rabbitmq publish success",
			slog.String("service", "sales"),
			slog.String("component", "rabbitmq"),
			slog.String("event", "publish"),
			slog.String("queue", queue),
			slog.Int("payload_bytes", len(body)),
		)
		return nil
	})
}

func (r *RabbitMQ) withChannel(fn func(*amqp.Channel) error) error {
	if !r.isAvailable() {
		slog.Warn("rabbitmq publish skipped",
			slog.String("service", "sales"),
			slog.String("component", "rabbitmq"),
			slog.String("reason", "unavailable"),
		)
		return ErrUnavailable
	}
	conn, channel, err := dialChannel(r.url)
	if err != nil {
		slog.Error("rabbitmq dial channel failed",
			slog.String("service", "sales"),
			slog.String("component", "rabbitmq"),
			slog.String("error", err.Error()),
		)
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
