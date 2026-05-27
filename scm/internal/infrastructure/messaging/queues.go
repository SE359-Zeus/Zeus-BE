package messaging

import (
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	PoolQueue     = "system.deficit.pool"
	ReservedQueue = "system.deficit.reserved"
	DLXExchange   = "system.dlx"
	DLXQueue      = "system.deficit.dlx"
	AuditQueue    = "system.audit.log"
)

func setupQueues(channel *amqp.Channel) error {
	if channel == nil {
		return ErrUnavailable
	}
	if err := channel.ExchangeDeclare(DLXExchange, "direct", true, false, false, false, nil); err != nil {
		return fmt.Errorf("failed to declare DLX exchange: %w", err)
	}
	if _, err := channel.QueueDeclare(PoolQueue, true, false, false, false, nil); err != nil {
		return fmt.Errorf("failed to declare pool queue: %w", err)
	}
	if _, err := channel.QueueDeclare(DLXQueue, true, false, false, false, nil); err != nil {
		return fmt.Errorf("failed to declare DLX queue: %w", err)
	}
	if err := channel.QueueBind(DLXQueue, DLXQueue, DLXExchange, false, nil); err != nil {
		return fmt.Errorf("failed to bind DLX queue: %w", err)
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
