package messaging

import (
	"context"
	"encoding/json"
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

func (c *Connection) Close() {
	if c.channel != nil {
		c.channel.Close()
	}
	if c.conn != nil {
		c.conn.Close()
	}
}

func (c *Connection) Channel() *amqp.Channel {
	return c.channel
}

func (c *Connection) GetFromPool(autoAck bool) (amqp.Delivery, bool, error) {
	return c.channel.Get(PoolQueue, autoAck)
}

func (c *Connection) PublishToPool(ctx context.Context, msg DeficitMessage) error {
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

func (c *Connection) PublishToReserved(ctx context.Context, msg DeficitMessage) error {
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

func (c *Connection) Ack(tag uint64) error {
	return c.channel.Ack(tag, false)
}

func (c *Connection) Nack(tag uint64, requeue bool) error {
	return c.channel.Nack(tag, false, requeue)
}

func (c *Connection) QueueSize(queue string) (int, error) {
	q, err := c.channel.QueueInspect(queue)
	if err != nil {
		return 0, err
	}
	return q.Messages, nil
}

func (c *Connection) SetupQueues() error {
	if _, err := c.channel.QueueDeclare(
		PoolQueue, true, false, false, false, nil,
	); err != nil {
		return fmt.Errorf("failed to declare pool queue: %w", err)
	}
	if _, err := c.channel.QueueDeclare(
		DLXQueue, true, false, false, false, nil,
	); err != nil {
		return fmt.Errorf("failed to declare DLX queue: %w", err)
	}
	if _, err := c.channel.QueueDeclare(
		ReservedQueue, true, false, false, false,
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

func Stats(ctx context.Context, url string) (*DeficitPoolStats, error) {
	pool, err := QueueSize(ctx, url, PoolQueue)
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

func EnsureQueues(url string) error {
	c, err := Dial(url)
	if err != nil {
		return err
	}
	defer c.Close()
	return c.SetupQueues()
}

func StartExpiryReconciler(ctx context.Context, url string, interval time.Duration) {
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
