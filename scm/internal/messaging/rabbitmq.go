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
	PoolQueue     = "scm.deficit.pool"
	ReservedQueue = "scm.deficit.reserved"
	DLXExchange   = "scm.dlx"
	DLXQueue      = "scm.deficit.dlx"
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
	conn    *amqp.Connection
	channel *amqp.Channel
}

type Connection struct {
	conn    *amqp.Connection
	channel *amqp.Channel
}

func NewRabbitMQ(url string) (*RabbitMQ, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}
	channel, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}
	rabbit := &RabbitMQ{conn: conn, channel: channel}
	if err := rabbit.setupQueues(); err != nil {
		rabbit.Close()
		return nil, err
	}
	return rabbit, nil
}

func Dial(url string) (*Connection, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}
	channel, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}
	return &Connection{conn: conn, channel: channel}, nil
}

func (r *RabbitMQ) setupQueues() error {
	if r == nil || r.channel == nil {
		return ErrUnavailable
	}
	if _, err := r.channel.QueueDeclare(PoolQueue, true, false, false, false, nil); err != nil {
		return fmt.Errorf("failed to declare pool queue: %w", err)
	}
	if _, err := r.channel.QueueDeclare(DLXQueue, true, false, false, false, nil); err != nil {
		return fmt.Errorf("failed to declare DLX queue: %w", err)
	}
	if _, err := r.channel.QueueDeclare(
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

func (r *RabbitMQ) PublishToPool(msg DeficitMessage) error {
	if r == nil || r.channel == nil {
		return ErrUnavailable
	}
	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return r.channel.PublishWithContext(context.Background(), "", PoolQueue, true, false, amqp.Publishing{
		ContentType: "application/json",
		Body:        body,
	})
}

func (r *RabbitMQ) ConsumeFromPool() (<-chan amqp.Delivery, error) {
	if r == nil || r.channel == nil {
		return nil, ErrUnavailable
	}
	return r.channel.Consume(PoolQueue, "", true, true, false, false, nil)
}

func (r *RabbitMQ) PublishToReserved(msg DeficitMessage) error {
	if r == nil || r.channel == nil {
		return ErrUnavailable
	}
	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return r.channel.PublishWithContext(context.Background(), "", ReservedQueue, true, false, amqp.Publishing{
		ContentType:  "application/json",
		Body:         body,
		Expiration:   fmt.Sprintf("%d", 30*60*1000),
		DeliveryMode: amqp.Persistent,
	})
}

func (r *RabbitMQ) ConsumeReserved() (<-chan amqp.Delivery, error) {
	if r == nil || r.channel == nil {
		return nil, ErrUnavailable
	}
	return r.channel.Consume(ReservedQueue, "", false, false, false, false, nil)
}

func (r *RabbitMQ) Ack(tag uint64) error {
	if r == nil || r.channel == nil {
		return ErrUnavailable
	}
	return r.channel.Ack(tag, false)
}

func (r *RabbitMQ) Nack(tag uint64, requeue bool) error {
	if r == nil || r.channel == nil {
		return ErrUnavailable
	}
	return r.channel.Nack(tag, false, requeue)
}

func (r *RabbitMQ) ConsumeDLX() (<-chan amqp.Delivery, error) {
	if r == nil || r.channel == nil {
		return nil, ErrUnavailable
	}
	return r.channel.Consume(DLXQueue, "", true, false, false, false, nil)
}

func (r *RabbitMQ) RequeueFromDLX(delivery amqp.Delivery) error {
	if r == nil || r.channel == nil {
		return ErrUnavailable
	}
	var msg DeficitMessage
	if err := json.Unmarshal(delivery.Body, &msg); err != nil {
		return err
	}
	return r.PublishToPool(msg)
}

func (r *RabbitMQ) QueueSize(queue string) (int, error) {
	if r == nil || r.channel == nil {
		return 0, ErrUnavailable
	}
	q, err := r.channel.QueueInspect(queue)
	if err != nil {
		return 0, err
	}
	return q.Messages, nil
}

func (r *RabbitMQ) Stats() (*DeficitPoolStats, error) {
	if r == nil || r.channel == nil {
		return nil, ErrUnavailable
	}
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
	if r == nil {
		return nil
	}
	return r.channel
}

func (r *RabbitMQ) Close() {
	if r == nil {
		return
	}
	if r.channel != nil {
		_ = r.channel.Close()
	}
	if r.conn != nil {
		_ = r.conn.Close()
	}
}

func (r *RabbitMQ) StartExpiryReconciler(interval time.Duration, stop <-chan struct{}) {
	if r == nil || r.channel == nil {
		return
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				msgs, err := r.ConsumeDLX()
				if err != nil {
					continue
				}
				for msg := range msgs {
					_ = r.RequeueFromDLX(msg)
				}
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

func (c *Connection) QueueSize(queue string) (int, error) {
	if c == nil || c.channel == nil {
		return 0, ErrUnavailable
	}
	q, err := c.channel.QueueInspect(queue)
	if err != nil {
		return 0, err
	}
	return q.Messages, nil
}

func (c *Connection) Ack(tag uint64) error {
	if c == nil || c.channel == nil {
		return ErrUnavailable
	}
	return c.channel.Ack(tag, false)
}

func (c *Connection) Nack(tag uint64, requeue bool) error {
	if c == nil || c.channel == nil {
		return ErrUnavailable
	}
	return c.channel.Nack(tag, false, requeue)
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
