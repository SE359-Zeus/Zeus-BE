package messaging

import (
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	AuditQueue             = "system.audit.log"
	DeficitPoolQueue       = "system.deficit.pool"
	DeficitReservedQueue   = "system.deficit.reserved"
	SalesOrderCreatedQueue = "sales.order.created"
	SalesOrderUpdatedQueue = "sales.order.updated"
)

// queueDef pairs a queue name with optional declaration arguments.
// Some queues (e.g. DeficitReservedQueue) already exist in RabbitMQ
// with specific args; the declaration must match or RabbitMQ returns 406.
type queueDef struct {
	name string
	args amqp.Table
}

func DeclareQueues(channel *amqp.Channel) error {
	if channel == nil {
		return ErrUnavailable
	}

	defs := []queueDef{
		{name: AuditQueue},
		{name: DeficitPoolQueue},
		{name: DeficitReservedQueue, args: amqp.Table{"x-message-ttl": int32(1_800_000)}},
		{name: SalesOrderCreatedQueue},
		{name: SalesOrderUpdatedQueue},
	}

	for _, d := range defs {
		if _, err := channel.QueueDeclare(d.name, true, false, false, false, d.args); err != nil {
			return fmt.Errorf("failed to declare queue %s: %w", d.name, err)
		}
	}

	return nil
}
