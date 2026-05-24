package messaging

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
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

func (m *DeficitMessage) FromDelivery(delivery amqp.Delivery) error {
	return json.Unmarshal(delivery.Body, m)
}

type DeficitPoolStats struct {
	PoolSize     int `json:"pool_size"`
	ReservedSize int `json:"reserved_size"`
	DLXSize      int `json:"dlx_size"`
}

type Connection struct {
	conn    *amqp.Connection
	channel *amqp.Channel
}

func Dial(url string) (*Connection, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("failed to dial RabbitMQ: %w", err)
	}
	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}
	return &Connection{conn: conn, channel: ch}, nil
}

func (r *RabbitMQ) setupQueues() error {
	if r == nil || r.channel == nil {
		return ErrUnavailable
	}
	_, err := r.channel.QueueDeclare(
		PoolQueue,
		true, false, false, false, nil,
	)
	if err != nil {
		return fmt.Errorf("failed to declare pool queue: %w", err)
	}
	_, err = r.channel.QueueDeclare(
		DLXQueue,
		true, false, false, false, nil,
	)
	if err != nil {
		return fmt.Errorf("failed to declare DLX queue: %w", err)
	}
}

func (c *Connection) Channel() *amqp.Channel {
	return c.channel
}

func (c *Connection) GetFromPool(autoAck bool) (amqp.Delivery, bool, error) {
	return c.channel.Get(PoolQueue, autoAck)
}

func (r *RabbitMQ) PublishToPool(msg DeficitMessage) error {
	if r == nil || r.channel == nil {
		return ErrUnavailable
	}
	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return c.channel.PublishWithContext(ctx,
		"", PoolQueue, true, false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)
}

func (r *RabbitMQ) ConsumeFromPool() (<-chan amqp.Delivery, error) {
	if r == nil || r.channel == nil {
		return nil, ErrUnavailable
	}
	return r.channel.Consume(
		PoolQueue, "", true, true, false, false, nil,
	)
}

func (r *RabbitMQ) PublishToReserved(msg DeficitMessage) error {
	if r == nil || r.channel == nil {
		return ErrUnavailable
	}
	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return c.channel.PublishWithContext(ctx,
		"", ReservedQueue, true, false,
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			Expiration:   fmt.Sprintf("%d", 30*60*1000),
			DeliveryMode: amqp.Persistent,
		},
	)
}

func (r *RabbitMQ) ConsumeReserved() (<-chan amqp.Delivery, error) {
	if r == nil || r.channel == nil {
		return nil, ErrUnavailable
	}
	return r.channel.Consume(
		ReservedQueue, "", false, false, false, false, nil,
	)
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
	return r.channel.Consume(
		DLXQueue, "", true, false, false, false, nil,
	)
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

func (m *DeficitMessage) FromDelivery(delivery amqp.Delivery) error {
	return json.Unmarshal(delivery.Body, m)
}

func (r *RabbitMQ) Close() {
	if r == nil {
		return
	}
	if r.channel != nil {
		r.channel.Close()
	}
	if r.conn != nil {
		r.conn.Close()
	}
	return nil
}

func PublishToPool(ctx context.Context, url string, msg DeficitMessage) error {
	c, err := Dial(url)
	if err != nil {
		return err
	}
	defer c.Close()
	return c.PublishToPool(ctx, msg)
}

func PublishToReserved(ctx context.Context, url string, msg DeficitMessage) error {
	c, err := Dial(url)
	if err != nil {
		return err
	}
	defer c.Close()
	return c.PublishToReserved(ctx, msg)
}

func QueueSize(ctx context.Context, url, queue string) (int, error) {
	c, err := Dial(url)
	if err != nil {
		return 0, err
	}
	defer c.Close()
	return c.QueueSize(queue)
}

func (r *RabbitMQ) Stats() (*DeficitPoolStats, error) {
	if r == nil || r.channel == nil {
		return nil, ErrUnavailable
	}
	pool, err := r.QueueSize(PoolQueue)
	if err != nil {
		return nil, err
	}
	reserved, err := QueueSize(ctx, url, ReservedQueue)
	if err != nil {
		return nil, err
	}
	dlx, err := QueueSize(ctx, url, DLXQueue)
	if err != nil {
		return nil, err
	}
	return &DeficitPoolStats{
		PoolSize:     pool,
		ReservedSize: reserved,
		DLXSize:      dlx,
	}, nil
}

func (r *RabbitMQ) Channel() *amqp.Channel {
	if r == nil {
		return nil
	}
	return r.channel
}

func (r *RabbitMQ) StartExpiryReconciler(interval time.Duration, stop <-chan struct{}) {
	if r == nil || r.channel == nil {
		return
	}
	go func() {
		for {
			if err := reconcileOnce(ctx, url); err != nil {
				log.Printf("expiry reconciler error: %v", err)
			}
			select {
			case <-time.After(interval):
			case <-ctx.Done():
				return
			}
		}
	}()
}

func reconcileOnce(ctx context.Context, url string) error {
	c, err := Dial(url)
	if err != nil {
		return err
	}
	defer c.Close()

	msgs, err := c.channel.Consume(
		DLXQueue, "", true, false, false, false, nil,
	)
	if err != nil {
		return fmt.Errorf("failed to consume DLX: %w", err)
	}

	timeout := time.After(5 * time.Second)
	for {
		select {
		case msg, ok := <-msgs:
			if !ok {
				return nil
			}
			var deficit DeficitMessage
			if err := json.Unmarshal(msg.Body, &deficit); err != nil {
				continue
			}
			body, _ := json.Marshal(deficit)
			if err := c.channel.PublishWithContext(ctx,
				"", PoolQueue, true, false,
				amqp.Publishing{
					ContentType: "application/json",
					Body:        body,
				},
			); err != nil {
				return err
			}
		case <-timeout:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
