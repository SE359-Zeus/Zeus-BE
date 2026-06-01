package consumer

import (
	"context"
	"encoding/json"
	"log"
	"strings"

	"zeus-system-service/internal/infrastructure/messaging"
	"zeus-system-service/internal/infrastructure/observability"
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

				// Preserve trace context from message headers when present so
				// system logs and traces keep the original trace_id/span_id.
				msgCtx := ctx
				if d.Headers != nil {
					// helper to coerce header values to string
					toStr := func(v interface{}) string {
						switch t := v.(type) {
						case string:
							return t
						case []byte:
							return string(t)
						default:
							return ""
						}
					}

					if v := toStr(d.Headers["trace_id"]); v != "" {
						msgCtx = observability.WithTraceID(msgCtx, v)
					} else if tp := toStr(d.Headers["traceparent"]); tp != "" {
						parts := strings.Split(tp, "-")
						if len(parts) == 4 && len(parts[1]) == 32 {
							msgCtx = observability.WithTraceID(msgCtx, parts[1])
						}
					}

					if v := toStr(d.Headers["span_id"]); v != "" {
						msgCtx = observability.WithSpanID(msgCtx, v)
					} else if tp := toStr(d.Headers["traceparent"]); tp != "" {
						parts := strings.Split(tp, "-")
						if len(parts) == 4 && len(parts[2]) == 16 {
							msgCtx = observability.WithSpanID(msgCtx, parts[2])
						}
					}
				}

				var req models.IngestAuditRequest
				if err := json.Unmarshal(d.Body, &req); err != nil {
					log.Printf("Error unmarshaling audit request: %v", err)
					_ = d.Nack(false, false)
					continue
				}

				if err := c.auditSvc.Ingest(msgCtx, req); err != nil {
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
