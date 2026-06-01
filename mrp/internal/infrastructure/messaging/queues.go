package messaging

import (
	"fmt"
	"log/slog"
	"strings"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	AuditQueue             = "system.audit.log"
	DeficitPoolQueue       = "system.deficit.pool"
	DeficitReservedQueue   = "system.deficit.reserved"
	SalesOrderCreatedQueue = "sales.order.created"
	SalesOrderUpdatedQueue = "sales.order.updated"
)

// DeclareQueues declares all queues the consumer needs.
// DeficitReservedQueue is declared on a separate temporary channel because
// it is owned by SCM which creates it with custom arguments (x-message-ttl,
// x-dead-letter-exchange, …). A mismatched QueueDeclare returns 406 and
// closes the channel — we must not let that kill the main channel.
func DeclareQueues(conn *amqp.Connection, channel *amqp.Channel) error {
	if conn == nil || channel == nil {
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

	// DeficitReservedQueue: declare on a throwaway channel so a 406
	// (args mismatch) cannot close the main consumer channel.
	if tmpCh, err := conn.Channel(); err == nil {
		if _, err := tmpCh.QueueDeclare(DeficitReservedQueue, true, false, false, false, nil); err != nil {
			if strings.Contains(err.Error(), "PRECONDITION_FAILED") {
				slog.Info("deficit.reserved queue already exists with different args, consuming as-is",
					slog.String("service", "mrp"),
					slog.String("component", "rabbitmq"),
					slog.String("queue", DeficitReservedQueue),
				)
			} else {
				slog.Warn("failed to declare deficit.reserved queue",
					slog.String("service", "mrp"),
					slog.String("component", "rabbitmq"),
					slog.String("error", err.Error()),
				)
			}
		}
		_ = tmpCh.Close()
	}

	return nil
}
