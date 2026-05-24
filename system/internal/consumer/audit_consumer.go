package consumer

import (
	"context"
	"encoding/json"
	"log"

	"zeus-system-service/internal/infrastructure/messaging"
	"zeus-system-service/internal/models"
	"zeus-system-service/internal/service"
)

type AuditConsumer struct {
	mqURL    string
	auditSvc service.AuditService
}

func NewAuditConsumer(mqURL string, auditSvc service.AuditService) *AuditConsumer {
	return &AuditConsumer{
		mqURL:    mqURL,
		auditSvc: auditSvc,
	}
}

func (c *AuditConsumer) Start(ctx context.Context) error {
	conn, err := messaging.Dial(c.mqURL)
	if err != nil {
		return err
	}

	msgs, err := conn.ConsumeAudit()
	if err != nil {
		return err
	}

	log.Println("Audit consumer started, listening on system.audit.log...")

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case d, ok := <-msgs:
				if !ok {
					log.Println("Audit message channel closed.")
					return
				}

				var req models.IngestAuditRequest
				if err := json.Unmarshal(d.Body, &req); err != nil {
					log.Printf("Error unmarshaling audit request: %v", err)
					_ = d.Nack(false, false)
					continue
				}

				if err := c.auditSvc.Ingest(ctx, req); err != nil {
					log.Printf("Error ingesting audit log via consumer: %v", err)
					_ = d.Nack(false, true) // Requeue
				} else {
					_ = d.Ack(false)
				}
			}
		}
	}()

	return nil
}
