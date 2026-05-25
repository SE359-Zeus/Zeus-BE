package messaging

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	PoolQueue     = "system.deficit.pool"
	ReservedQueue = "system.deficit.reserved"
	DLXExchange   = "system.dlx"
	DLXQueue      = "system.deficit.dlx"
	AuditQueue    = "system.audit.log"
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
	conn, channel, err := dialChannel(url)
	if err != nil {
		return nil, err
	}
	_ = channel.Close()
	_ = conn.Close()
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

func setupQueues(channel *amqp.Channel) error {
	if channel == nil {
		return ErrUnavailable
	}
	if _, err := channel.QueueDeclare(PoolQueue, true, false, false, false, nil); err != nil {
		return fmt.Errorf("failed to declare pool queue: %w", err)
	}
	if _, err := channel.QueueDeclare(DLXQueue, true, false, false, false, nil); err != nil {
		return fmt.Errorf("failed to declare DLX queue: %w", err)
	}
	if _, err := channel.QueueDeclare(AuditQueue, true, false, false, false, nil); err != nil {
		return fmt.Errorf("failed to declare audit queue: %w", err)
	}
	if _, err := channel.QueueDeclare(
		ReservedQueue,
		true,
		false,
		false,
		false,
		amqp.Table{
			"x-dead-letter-exchange":    DLXExchange,
			"x-dead-letter-routing-key": DLXQueue,
			"x-message-ttl":             int32(30 * 60 * 1000),
		},
	); err != nil {
		return fmt.Errorf("failed to declare reserved queue: %w", err)
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
	out := make(chan amqp.Delivery)
	go func() {
		defer close(out)
		defer channel.Close()
		defer conn.Close()
		for msg := range msgs {
			out <- msg
		}
	}()
	return out, nil
}

func (r *RabbitMQ) PublishToPool(msg DeficitMessage) error {
	return r.withChannel(func(channel *amqp.Channel) error {
		body, err := json.Marshal(msg)
		if err != nil {
			return err
		}
		return channel.PublishWithContext(context.Background(), "", PoolQueue, true, false, amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		})
	})
}

func (r *RabbitMQ) ConsumeFromPool() (<-chan amqp.Delivery, error) {
	return r.consume(PoolQueue, true, true)
}

func (r *RabbitMQ) PublishToReserved(msg DeficitMessage) error {
	return r.withChannel(func(channel *amqp.Channel) error {
		body, err := json.Marshal(msg)
		if err != nil {
			return err
		}
		return channel.PublishWithContext(context.Background(), "", ReservedQueue, true, false, amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			Expiration:   fmt.Sprintf("%d", 30*60*1000),
			DeliveryMode: amqp.Persistent,
		})
	})
}

func (r *RabbitMQ) ConsumeReserved() (<-chan amqp.Delivery, error) {
	return r.consume(ReservedQueue, false, false)
}

func (r *RabbitMQ) Ack(tag uint64) error {
	return r.withChannel(func(channel *amqp.Channel) error {
		return channel.Ack(tag, false)
	})
}

func (r *RabbitMQ) Nack(tag uint64, requeue bool) error {
	return r.withChannel(func(channel *amqp.Channel) error {
		return channel.Nack(tag, false, requeue)
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
	return c.channel.Get(PoolQueue, autoAck)
}

func (c *Connection) PublishToPool(ctx context.Context, msg DeficitMessage) error {
	if c == nil || c.channel == nil {
		return ErrUnavailable
	}
	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return c.channel.PublishWithContext(ctx, "", PoolQueue, true, false, amqp.Publishing{ContentType: "application/json", Body: body})
}

func (c *Connection) PublishToReserved(ctx context.Context, msg DeficitMessage) error {
	if c == nil || c.channel == nil {
		return ErrUnavailable
	}
	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return c.channel.PublishWithContext(ctx, "", ReservedQueue, true, false, amqp.Publishing{ContentType: "application/json", Body: body, Expiration: fmt.Sprintf("%d", 30*60*1000), DeliveryMode: amqp.Persistent})
}

func (r *RabbitMQ) ConsumeAudit() (<-chan amqp.Delivery, error) {
	return r.consume(AuditQueue, false, false)
}

func (c *Connection) ConsumeAudit() (<-chan amqp.Delivery, error) {
	if c == nil || c.channel == nil {
		return nil, ErrUnavailable
	}
	return c.channel.Consume(AuditQueue, "", false, false, false, false, nil)
}

func (c *Connection) PublishToAudit(ctx context.Context, msg any) error {
	if c == nil || c.channel == nil {
		return ErrUnavailable
	}
	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return c.channel.PublishWithContext(ctx, "", AuditQueue, true, false, amqp.Publishing{
		ContentType:  "application/json",
		Body:         body,
		DeliveryMode: amqp.Persistent,
	})
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
