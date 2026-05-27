package messaging

import (
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	AuditQueue             = "system.audit.log"
	DeficitPoolQueue       = "system.deficit.pool"
	SalesOrderCreatedQueue = "sales.order.created"
	SalesOrderUpdatedQueue = "sales.order.updated"
)

func DeclareQueues(channel *amqp.Channel) error {
	if channel == nil {
		return ErrUnavailable
	}

	for _, queue := range []string{
		AuditQueue,
		DeficitPoolQueue,
		SalesOrderCreatedQueue,
		SalesOrderUpdatedQueue,
	} {
		if _, err := channel.QueueDeclare(queue, true, false, false, false, nil); err != nil {
			return fmt.Errorf("failed to declare queue %s: %w", queue, err)
		}
	}

	return nil
}
