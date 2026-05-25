package messaging

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"sync"

	amqp "github.com/rabbitmq/amqp091-go"
)

var ErrUnavailable = errors.New("rabbitmq unavailable")

const AuditQueue = "system.audit.log"

type RabbitMQ struct {
	url       string
	available bool
	mu        sync.RWMutex
}

func NewRabbitMQ(url string) *RabbitMQ {
	r := &RabbitMQ{url: url}
	if url == "" {
		log.Println("RabbitMQ disabled: no URL configured")
		return r
	}

	conn, err := amqp.Dial(url)
	if err != nil {
		log.Printf("Warning: RabbitMQ connection failed at %s: %v", url, err)
		return r
	}
	_ = conn.Close()

	log.Printf("RabbitMQ connection successful at %s", url)
	r.available = true
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

func (r *RabbitMQ) Close() {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.available = false
}

func (r *RabbitMQ) withChannel(fn func(*amqp.Channel) error) error {
	if !r.isAvailable() {
		return ErrUnavailable
	}
	conn, err := amqp.Dial(r.url)
	if err != nil {
		return err
	}
	defer conn.Close()

	channel, err := conn.Channel()
	if err != nil {
		return err
	}
	defer channel.Close()

	return fn(channel)
}

func (r *RabbitMQ) DeclareQueue(queue string, durable bool) error {
	return r.withChannel(func(channel *amqp.Channel) error {
		_, err := channel.QueueDeclare(queue, durable, false, false, false, nil)
		return err
	})
}

func (r *RabbitMQ) PublishJSON(queue string, payload any) error {
	return r.withChannel(func(channel *amqp.Channel) error {
		if _, err := channel.QueueDeclare(queue, true, false, false, false, nil); err != nil {
			return err
		}
		body, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		return channel.PublishWithContext(context.Background(), "", queue, true, false, amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent,
		})
	})
}
