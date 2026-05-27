package messaging

import amqp "github.com/rabbitmq/amqp091-go"

const (
	OrderCreatedQueue    = "sales.order.created"
	OrderUpdatedQueue    = "sales.order.updated"
	OrderAllocatedQueue  = "sales.order.allocated"
	OrderCancelledQueue  = "sales.order.cancelled"
	ClientUpdatedQueue   = "sales.client.updated"
	AuditQueue           = "system.audit.log"
	FulfillmentQueued    = "sales.fulfillment.queued"
	FulfillmentProcessed = "sales.fulfillment.processed"
)

func declareQueues(channel *amqp.Channel) error {
	for _, queue := range []string{
		OrderCreatedQueue,
		OrderUpdatedQueue,
		OrderAllocatedQueue,
		OrderCancelledQueue,
		ClientUpdatedQueue,
		AuditQueue,
		FulfillmentQueued,
		FulfillmentProcessed,
	} {
		if _, err := channel.QueueDeclare(queue, true, false, false, false, nil); err != nil {
			return err
		}
	}
	return nil
}
