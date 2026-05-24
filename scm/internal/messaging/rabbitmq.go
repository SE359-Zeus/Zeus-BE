package messaging

import (
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

type RabbitMQ struct {
	conn    *amqp.Connection
	channel *amqp.Channel
}

func NewRabbitMQ(url string) (*RabbitMQ, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}
	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}
	rmq := &RabbitMQ{conn: conn, channel: ch}
	if err := rmq.setupQueues(); err != nil {
		rmq.Close()
		return nil, err
	}
	return rmq, nil
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
	_, err = r.channel.QueueDeclare(
		ReservedQueue,
		true, false, false, false,
		amqp.Table{
			"x-dead-letter-exchange":    DLXExchange,
			"x-dead-letter-routing-key": DLXQueue,
			"x-message-ttl":             int32(30 * 60 * 1000), // 30 min
		},
	)
	if err != nil {
		return fmt.Errorf("failed to declare reserved queue: %w", err)
	}
	_, err = r.channel.QueueDeclare(
		"", false, false, true, false, nil,
	)
	return err
}

func (r *RabbitMQ) PublishToPool(msg DeficitMessage) error {
	if r == nil || r.channel == nil {
		return ErrUnavailable
	}
	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return r.channel.Publish(
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
	return r.channel.Publish(
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
}

type DeficitPoolStats struct {
	PoolSize     int `json:"pool_size"`
	ReservedSize int `json:"reserved_size"`
	DLXSize      int `json:"dlx_size"`
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
