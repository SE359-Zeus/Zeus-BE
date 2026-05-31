package messaging

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"
	"zeus-scm-service/internal/infrastructure/observability"

	amqp "github.com/rabbitmq/amqp091-go"
)

var ErrUnavailable = errors.New("rabbitmq unavailable")

type DeficitMessage struct {
	SKU     string `json:"sku"`
	Qty     int    `json:"qty"`
	OrderID string `json:"order_id"`
}

type DeficitPoolStats struct {
	PoolSize     int `json:"pool_size"`
	ReservedSize int `json:"reserved_size"`
	DLXSize      int `json:"dlx_size"`
}

type RabbitMQ struct {
	url string
}

type Connection struct {
	conn    *amqp.Connection
	channel *amqp.Channel
}

func NewRabbitMQ(url string) (*RabbitMQ, error) {
	if url == "" {
		slog.Warn("rabbitmq unavailable",
			slog.String("service", "scm"),
			slog.String("component", "rabbitmq"),
			slog.String("reason", "empty_url"),
		)
		return nil, ErrUnavailable
	}
	conn, channel, err := dialChannel(url)
	if err != nil {
		slog.Warn("rabbitmq connection failed",
			slog.String("service", "scm"),
			slog.String("component", "rabbitmq"),
			slog.String("url", url),
			slog.String("error", err.Error()),
		)
		return nil, err
	}
	_ = channel.Close()
	_ = conn.Close()
	slog.Info("rabbitmq connected",
		slog.String("service", "scm"),
		slog.String("component", "rabbitmq"),
		slog.String("url", url),
	)
	return &RabbitMQ{url: url}, nil
}

func Dial(url string) (*Connection, error) {
	conn, channel, err := dialChannel(url)
	if err != nil {
		return nil, err
	}
	return &Connection{conn: conn, channel: channel}, nil
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
	if err := setupQueues(channel); err != nil {
		_ = channel.Close()
		_ = conn.Close()
		return nil, nil, err
	}
	return conn, channel, nil
}

func (r *RabbitMQ) withChannel(fn func(*amqp.Channel) error) error {
	if r == nil || r.url == "" {
		slog.Warn("rabbitmq unavailable",
			slog.String("service", "scm"),
			slog.String("component", "rabbitmq"),
			slog.String("reason", "empty_client_or_url"),
		)
		return ErrUnavailable
	}
	conn, channel, err := dialChannel(r.url)
	if err != nil {
		slog.Error("rabbitmq dial failed",
			slog.String("service", "scm"),
			slog.String("component", "rabbitmq"),
			slog.String("error", err.Error()),
		)
		return err
	}
	defer conn.Close()
	defer channel.Close()
	return fn(channel)
}

func (r *RabbitMQ) consume(queue string, autoAck bool, exclusive bool) (<-chan amqp.Delivery, error) {
	if r == nil || r.url == "" {
		return nil, ErrUnavailable
	}
	conn, channel, err := dialChannel(r.url)
	if err != nil {
		return nil, err
	}
	msgs, err := channel.Consume(queue, "", autoAck, exclusive, false, false, nil)
	if err != nil {
		_ = channel.Close()
		_ = conn.Close()
		return nil, err
	}
	slog.Info("rabbitmq consumer subscribed",
		slog.String("service", "scm"),
		slog.String("component", "rabbitmq"),
		slog.String("event", "consume_subscribed"),
		slog.String("queue", queue),
		slog.Bool("auto_ack", autoAck),
		slog.Bool("exclusive", exclusive),
	)
	out := make(chan amqp.Delivery)
	go func() {
		defer close(out)
		defer channel.Close()
		defer conn.Close()
		for msg := range msgs {
			slog.Info("rabbitmq message received",
				slog.String("service", "scm"),
				slog.String("component", "rabbitmq"),
				slog.String("event", "consume_received"),
				slog.String("queue", queue),
				slog.Uint64("delivery_tag", msg.DeliveryTag),
				slog.Int("payload_bytes", len(msg.Body)),
			)
			out <- msg
		}
	}()
	return out, nil
}

func (r *RabbitMQ) PublishToPool(msg DeficitMessage) error {
	return r.withChannel(func(channel *amqp.Channel) error {
		body, err := json.Marshal(msg)
		if err != nil {
			slog.Error("rabbitmq publish marshal failed",
				slog.String("service", "scm"),
				slog.String("component", "rabbitmq"),
				slog.String("event", "publish"),
				slog.String("queue", PoolQueue),
				slog.String("error", err.Error()),
			)
			return err
		}
		err = channel.PublishWithContext(context.Background(), "", PoolQueue, true, false, amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		})
		if err != nil {
			slog.Error("rabbitmq publish failed",
				slog.String("service", "scm"),
				slog.String("component", "rabbitmq"),
				slog.String("event", "publish"),
				slog.String("queue", PoolQueue),
				slog.String("error", err.Error()),
			)
			return err
		}
		slog.Info("rabbitmq publish success",
			slog.String("service", "scm"),
			slog.String("component", "rabbitmq"),
			slog.String("event", "publish"),
			slog.String("queue", PoolQueue),
			slog.Int("payload_bytes", len(body)),
		)
		return nil
	})
}

func (r *RabbitMQ) ConsumeFromPool() (<-chan amqp.Delivery, error) {
	return r.consume(PoolQueue, true, true)
}

func (r *RabbitMQ) PublishToReserved(msg DeficitMessage) error {
	return r.withChannel(func(channel *amqp.Channel) error {
		body, err := json.Marshal(msg)
		if err != nil {
			slog.Error("rabbitmq publish marshal failed",
				slog.String("service", "scm"),
				slog.String("component", "rabbitmq"),
				slog.String("event", "publish"),
				slog.String("queue", ReservedQueue),
				slog.String("error", err.Error()),
			)
			return err
		}
		err = channel.PublishWithContext(context.Background(), "", ReservedQueue, true, false, amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			Expiration:   fmt.Sprintf("%d", 30*60*1000),
			DeliveryMode: amqp.Persistent,
		})
		if err != nil {
			slog.Error("rabbitmq publish failed",
				slog.String("service", "scm"),
				slog.String("component", "rabbitmq"),
				slog.String("event", "publish"),
				slog.String("queue", ReservedQueue),
				slog.String("error", err.Error()),
			)
			return err
		}
		slog.Info("rabbitmq publish success",
			slog.String("service", "scm"),
			slog.String("component", "rabbitmq"),
			slog.String("event", "publish"),
			slog.String("queue", ReservedQueue),
			slog.Int("payload_bytes", len(body)),
		)
		return nil
	})
}

func (r *RabbitMQ) PublishToAudit(msg any) error {
	return r.withChannel(func(channel *amqp.Channel) error {
		body, err := json.Marshal(msg)
		if err != nil {
			slog.Error("rabbitmq publish marshal failed",
				slog.String("service", "scm"),
				slog.String("component", "rabbitmq"),
				slog.String("event", "publish"),
				slog.String("queue", AuditQueue),
				slog.String("error", err.Error()),
			)
			return err
		}
		err = channel.PublishWithContext(context.Background(), "", AuditQueue, true, false, amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent,
		})
		if err != nil {
			slog.Error("rabbitmq publish failed",
				slog.String("service", "scm"),
				slog.String("component", "rabbitmq"),
				slog.String("event", "publish"),
				slog.String("queue", AuditQueue),
				slog.String("error", err.Error()),
			)
			return err
		}
		slog.Info("rabbitmq publish success",
			slog.String("service", "scm"),
			slog.String("component", "rabbitmq"),
			slog.String("event", "publish"),
			slog.String("queue", AuditQueue),
			slog.Int("payload_bytes", len(body)),
		)
		return nil
	})
}

func (r *RabbitMQ) ConsumeReserved() (<-chan amqp.Delivery, error) {
	return r.consume(ReservedQueue, false, false)
}

func (r *RabbitMQ) Ack(tag uint64) error {
	return r.withChannel(func(channel *amqp.Channel) error {
		err := channel.Ack(tag, false)
		if err != nil {
			slog.Error("rabbitmq ack failed",
				slog.String("service", "scm"),
				slog.String("component", "rabbitmq"),
				slog.Uint64("delivery_tag", tag),
				slog.String("error", err.Error()),
			)
			return err
		}
		slog.Info("rabbitmq ack success",
			slog.String("service", "scm"),
			slog.String("component", "rabbitmq"),
			slog.Uint64("delivery_tag", tag),
		)
		return nil
	})
}

func (r *RabbitMQ) Nack(tag uint64, requeue bool) error {
	return r.withChannel(func(channel *amqp.Channel) error {
		err := channel.Nack(tag, false, requeue)
		if err != nil {
			slog.Error("rabbitmq nack failed",
				slog.String("service", "scm"),
				slog.String("component", "rabbitmq"),
				slog.Uint64("delivery_tag", tag),
				slog.Bool("requeue", requeue),
				slog.String("error", err.Error()),
			)
			return err
		}
		slog.Info("rabbitmq nack success",
			slog.String("service", "scm"),
			slog.String("component", "rabbitmq"),
			slog.Uint64("delivery_tag", tag),
			slog.Bool("requeue", requeue),
		)
		return nil
	})
}

func (r *RabbitMQ) ConsumeDLX() (<-chan amqp.Delivery, error) {
	return r.consume(DLXQueue, true, false)
}

func (r *RabbitMQ) RequeueFromDLX(delivery amqp.Delivery) error {
	var msg DeficitMessage
	if err := json.Unmarshal(delivery.Body, &msg); err != nil {
		return err
	}
	return r.PublishToPool(msg)
}

func (r *RabbitMQ) QueueSize(queue string) (int, error) {
	var size int
	err := r.withChannel(func(channel *amqp.Channel) error {
		q, err := channel.QueueInspect(queue)
		if err != nil {
			return err
		}
		size = q.Messages
		return nil
	})
	return size, err
}

func (r *RabbitMQ) Stats() (*DeficitPoolStats, error) {
	pool, err := r.QueueSize(PoolQueue)
	if err != nil {
		return nil, err
	}
	reserved, err := r.QueueSize(ReservedQueue)
	if err != nil {
		return nil, err
	}
	dlx, err := r.QueueSize(DLXQueue)
	if err != nil {
		return nil, err
	}
	return &DeficitPoolStats{PoolSize: pool, ReservedSize: reserved, DLXSize: dlx}, nil
}

func (r *RabbitMQ) Channel() *amqp.Channel {
	return nil
}

func (r *RabbitMQ) Close() {
}

func (r *RabbitMQ) StartExpiryReconciler(interval time.Duration, stop <-chan struct{}) {
	if r == nil || r.url == "" {
		return
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				_ = r.withChannel(func(channel *amqp.Channel) error {
					for {
						delivery, ok, err := channel.Get(DLXQueue, false)
						if err != nil {
							return err
						}
						if !ok {
							return nil
						}
						var msg DeficitMessage
						if err := json.Unmarshal(delivery.Body, &msg); err != nil {
							_ = channel.Nack(delivery.DeliveryTag, false, false)
							continue
						}
						body, err := json.Marshal(msg)
						if err != nil {
							_ = channel.Nack(delivery.DeliveryTag, false, true)
							continue
						}
						if err := channel.PublishWithContext(context.Background(), "", PoolQueue, true, false, amqp.Publishing{ContentType: "application/json", Body: body}); err != nil {
							_ = channel.Nack(delivery.DeliveryTag, false, true)
							continue
						}
						if err := channel.Ack(delivery.DeliveryTag, false); err != nil {
							return err
						}
					}
				})
			case <-stop:
				return
			}
		}
	}()
}

func (c *Connection) Channel() *amqp.Channel {
	if c == nil {
		return nil
	}
	return c.channel
}

func (c *Connection) GetFromPool(autoAck bool) (amqp.Delivery, bool, error) {
	if c == nil || c.channel == nil {
		return amqp.Delivery{}, false, ErrUnavailable
	}
	msg, ok, err := c.channel.Get(PoolQueue, autoAck)
	if err != nil {
		slog.Error("rabbitmq consume failed",
			slog.String("service", "scm"),
			slog.String("component", "rabbitmq"),
			slog.String("event", "consume_get"),
			slog.String("queue", PoolQueue),
			slog.String("error", err.Error()),
		)
		return amqp.Delivery{}, false, err
	}
	if ok {
		slog.Info("rabbitmq message received",
			slog.String("service", "scm"),
			slog.String("component", "rabbitmq"),
			slog.String("event", "consume_get"),
			slog.String("queue", PoolQueue),
			slog.Uint64("delivery_tag", msg.DeliveryTag),
			slog.Int("payload_bytes", len(msg.Body)),
		)
	}
	return msg, ok, nil
}

func (c *Connection) PeekPool() ([]DeficitMessage, error) {
	if c == nil || c.channel == nil {
		return nil, ErrUnavailable
	}
	var msgs []DeficitMessage
	var deliveries []uint64
	for {
		msg, ok, err := c.channel.Get(PoolQueue, false)
		if err != nil {
			for _, tag := range deliveries {
				_ = c.channel.Nack(tag, false, true)
			}
			return nil, err
		}
		if !ok {
			break
		}
		deliveries = append(deliveries, msg.DeliveryTag)
		var d DeficitMessage
		if err := json.Unmarshal(msg.Body, &d); err == nil {
			msgs = append(msgs, d)
		}
	}
	for _, tag := range deliveries {
		_ = c.channel.Nack(tag, false, true)
	}
	return msgs, nil
}



func injectTraceHeaders(ctx context.Context, headers amqp.Table) amqp.Table {
	if headers == nil {
		headers = amqp.Table{}
	}
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
	return headers
}

func (c *Connection) PublishToPool(ctx context.Context, msg DeficitMessage) error {
	if c == nil || c.channel == nil {
		return ErrUnavailable
	}
	body, err := json.Marshal(msg)
	if err != nil {
		slog.Error("rabbitmq publish marshal failed",
			slog.String("service", "scm"),
			slog.String("component", "rabbitmq"),
			slog.String("event", "publish"),
			slog.String("queue", PoolQueue),
			slog.String("error", err.Error()),
		)
		return err
	}
	err = c.channel.PublishWithContext(ctx, "", PoolQueue, true, false, amqp.Publishing{
		ContentType: "application/json",
		Body:        body,
		Headers:     injectTraceHeaders(ctx, nil),
	})
	if err != nil {
		slog.Error("rabbitmq publish failed",
			slog.String("service", "scm"),
			slog.String("component", "rabbitmq"),
			slog.String("event", "publish"),
			slog.String("queue", PoolQueue),
			slog.String("error", err.Error()),
		)
		return err
	}
	slog.Info("rabbitmq publish success",
		slog.String("service", "scm"),
		slog.String("component", "rabbitmq"),
		slog.String("event", "publish"),
		slog.String("queue", PoolQueue),
		slog.Int("payload_bytes", len(body)),
	)
	return nil
}

func (c *Connection) PublishToReserved(ctx context.Context, msg DeficitMessage) error {
	if c == nil || c.channel == nil {
		return ErrUnavailable
	}
	body, err := json.Marshal(msg)
	if err != nil {
		slog.Error("rabbitmq publish marshal failed",
			slog.String("service", "scm"),
			slog.String("component", "rabbitmq"),
			slog.String("event", "publish"),
			slog.String("queue", ReservedQueue),
			slog.String("error", err.Error()),
		)
		return err
	}
	err = c.channel.PublishWithContext(ctx, "", ReservedQueue, true, false, amqp.Publishing{
		ContentType:  "application/json",
		Body:         body,
		Expiration:   fmt.Sprintf("%d", 30*60*1000),
		DeliveryMode: amqp.Persistent,
		Headers:      injectTraceHeaders(ctx, nil),
	})
	if err != nil {
		slog.Error("rabbitmq publish failed",
			slog.String("service", "scm"),
			slog.String("component", "rabbitmq"),
			slog.String("event", "publish"),
			slog.String("queue", ReservedQueue),
			slog.String("error", err.Error()),
		)
		return err
	}
	slog.Info("rabbitmq publish success",
		slog.String("service", "scm"),
		slog.String("component", "rabbitmq"),
		slog.String("event", "publish"),
		slog.String("queue", ReservedQueue),
		slog.Int("payload_bytes", len(body)),
	)
	return nil
}

func (r *RabbitMQ) ConsumeAudit() (<-chan amqp.Delivery, error) {
	return r.consume(AuditQueue, false, false)
}

func (c *Connection) ConsumeAudit() (<-chan amqp.Delivery, error) {
	if c == nil || c.channel == nil {
		return nil, ErrUnavailable
	}
	slog.Info("rabbitmq consumer subscribed",
		slog.String("service", "scm"),
		slog.String("component", "rabbitmq"),
		slog.String("event", "consume_subscribed"),
		slog.String("queue", AuditQueue),
		slog.Bool("auto_ack", false),
		slog.Bool("exclusive", false),
	)
	return c.channel.Consume(AuditQueue, "", false, false, false, false, nil)
}

func (c *Connection) PublishToAudit(ctx context.Context, msg any) error {
	if c == nil || c.channel == nil {
		return ErrUnavailable
	}
	body, err := json.Marshal(msg)
	if err != nil {
		slog.Error("rabbitmq publish marshal failed",
			slog.String("service", "scm"),
			slog.String("component", "rabbitmq"),
			slog.String("event", "publish"),
			slog.String("queue", AuditQueue),
			slog.String("error", err.Error()),
		)
		return err
	}
	err = c.channel.PublishWithContext(ctx, "", AuditQueue, true, false, amqp.Publishing{
		ContentType:  "application/json",
		Body:         body,
		DeliveryMode: amqp.Persistent,
		Headers:      injectTraceHeaders(ctx, nil),
	})
	if err != nil {
		slog.Error("rabbitmq publish failed",
			slog.String("service", "scm"),
			slog.String("component", "rabbitmq"),
			slog.String("event", "publish"),
			slog.String("queue", AuditQueue),
			slog.String("error", err.Error()),
		)
		return err
	}
	slog.Info("rabbitmq publish success",
		slog.String("service", "scm"),
		slog.String("component", "rabbitmq"),
		slog.String("event", "publish"),
		slog.String("queue", AuditQueue),
		slog.Int("payload_bytes", len(body)),
	)
	return nil
}

func (c *Connection) Ack(tag uint64) error {
	if c == nil || c.channel == nil {
		return ErrUnavailable
	}
	err := c.channel.Ack(tag, false)
	if err != nil {
		slog.Error("rabbitmq ack failed",
			slog.String("service", "scm"),
			slog.String("component", "rabbitmq"),
			slog.Uint64("delivery_tag", tag),
			slog.String("error", err.Error()),
		)
		return err
	}
	slog.Info("rabbitmq ack success",
		slog.String("service", "scm"),
		slog.String("component", "rabbitmq"),
		slog.Uint64("delivery_tag", tag),
	)
	return nil
}

func (c *Connection) Nack(tag uint64, requeue bool) error {
	if c == nil || c.channel == nil {
		return ErrUnavailable
	}
	err := c.channel.Nack(tag, false, requeue)
	if err != nil {
		slog.Error("rabbitmq nack failed",
			slog.String("service", "scm"),
			slog.String("component", "rabbitmq"),
			slog.Uint64("delivery_tag", tag),
			slog.Bool("requeue", requeue),
			slog.String("error", err.Error()),
		)
		return err
	}
	slog.Info("rabbitmq nack success",
		slog.String("service", "scm"),
		slog.String("component", "rabbitmq"),
		slog.Uint64("delivery_tag", tag),
		slog.Bool("requeue", requeue),
	)
	return nil
}

func (c *Connection) Close() {
	if c == nil {
		return
	}
	if c.channel != nil {
		_ = c.channel.Close()
	}
	if c.conn != nil {
		_ = c.conn.Close()
	}
}
