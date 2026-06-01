package consumer

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"time"

	"zeus-scm-service/internal/exception"
	"zeus-scm-service/internal/infrastructure/messaging"
	"zeus-scm-service/internal/infrastructure/observability"

	amqp "github.com/rabbitmq/amqp091-go"
)

var ErrInsufficientDeficit = exception.ErrInsufficientDeficit

type DeficitLockManager struct {
	mqURL string
}

func NewDeficitLockManager(mqURL string) *DeficitLockManager {
	return &DeficitLockManager{mqURL: mqURL}
}

type fetchedMessage struct {
	deliveryTag uint64
	sku         string
	qty         int
	body        []byte
	headers     amqp.Table
}

// LockDeficit locks the required quantity of deficit messages for the given SKU.
func (m *DeficitLockManager) LockDeficit(ctx context.Context, sku string, qty int) error {
	conn, err := messaging.Dial(m.mqURL)
	if err != nil {
		slog.Warn("rabbitmq unavailable; proceeding without deficit reservation",
			slog.String("service", "scm"),
			slog.String("component", "purchase_order"),
			slog.String("event", "dependency_unavailable"),
			slog.Any("error", err),
		)
		return err
	}
	defer conn.Close()

	done := make(chan struct{}, 1)
	var consumeErr error

	go func() {
		defer func() { close(done) }()

		var matched []fetchedMessage
		var unmatched []fetchedMessage
		accumulatedQty := 0

		for {
			msg, ok, err := conn.GetFromPool(false)
			if err != nil {
				for _, fm := range matched {
					_ = fm.requeueWithBackoff(ctx, conn)
				}
				for _, fm := range unmatched {
					_ = fm.requeueWithBackoff(ctx, conn)
				}
				consumeErr = err
				return
			}
			if !ok {
				break
			}

			var d messaging.DeficitMessage
			if err := json.Unmarshal(msg.Body, &d); err != nil {
				// Discard malformed messages with immediate ack to clear queue
				_ = conn.Ack(msg.DeliveryTag)
				continue
			}

			fm := fetchedMessage{
				deliveryTag: msg.DeliveryTag,
				sku:         d.SKU,
				qty:         d.Qty,
				body:        msg.Body,
				headers:     msg.Headers,
			}

			if d.SKU == sku {
				matched = append(matched, fm)
				accumulatedQty += d.Qty
				if accumulatedQty >= qty {
					break
				}
			} else {
				unmatched = append(unmatched, fm)
			}
		}

		if accumulatedQty >= qty {
			for _, fm := range matched {
				_ = conn.Ack(fm.deliveryTag)
			}
			for _, fm := range unmatched {
				_ = fm.requeueWithBackoff(ctx, conn)
			}

			if accumulatedQty > qty {
				remainderMsg := messaging.DeficitMessage{
					SKU: sku,
					Qty: accumulatedQty - qty,
				}
				_ = conn.PublishToPool(ctx, remainderMsg)
			}

			msgCtx := ctx
			if len(matched) > 0 {
				lastMsg := matched[len(matched)-1]
				msgTraceID := ""
				msgSpanID := ""
				if lastMsg.headers != nil {
					if tpVal, ok := lastMsg.headers["traceparent"]; ok {
						var tpStr string
						switch v := tpVal.(type) {
						case string:
							tpStr = v
						case []byte:
							tpStr = string(v)
						}
						if tpStr != "" {
							parts := strings.Split(tpStr, "-")
							if len(parts) == 4 && len(parts[1]) == 32 {
								msgTraceID = parts[1]
							}
						}
					}
					if msgTraceID == "" {
						if tidVal, ok := lastMsg.headers["trace_id"]; ok {
							switch v := tidVal.(type) {
							case string:
								msgTraceID = v
							case []byte:
								msgTraceID = string(v)
							}
						}
					}
					if msgSpanID == "" {
						if sidVal, ok := lastMsg.headers["span_id"]; ok {
							switch v := sidVal.(type) {
							case string:
								msgSpanID = v
							case []byte:
								msgSpanID = string(v)
							}
						}
					}
				}
				if msgTraceID != "" {
					msgCtx = observability.WithTraceID(ctx, msgTraceID)
					if msgSpanID != "" {
						msgCtx = observability.WithSpanID(msgCtx, msgSpanID)
					}
				}
			}

			reservedMsg := messaging.DeficitMessage{
				SKU: sku,
				Qty: qty,
			}
			if err := conn.PublishToReserved(msgCtx, reservedMsg); err != nil {
				consumeErr = err
				return
			}
		} else {
			for _, fm := range matched {
				_ = fm.requeueWithBackoff(ctx, conn)
			}
			for _, fm := range unmatched {
				_ = fm.requeueWithBackoff(ctx, conn)
			}
			consumeErr = ErrInsufficientDeficit
		}
	}()

	select {
	case <-done:
		if consumeErr != nil {
			return consumeErr
		}
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(5 * time.Second):
		return ErrInsufficientDeficit
	}

	return nil
}

// requeueWithBackoff requeues a message with exponential backoff and dead-lettering.
func (fm *fetchedMessage) requeueWithBackoff(ctx context.Context, conn *messaging.Connection) error {
	var retryCount int32 = 0
	if fm.headers == nil {
		fm.headers = amqp.Table{}
	}

	if val, ok := fm.headers["x-retry-count"]; ok {
		switch v := val.(type) {
		case int32:
			retryCount = v
		case int64:
			retryCount = int32(v)
		case float64:
			retryCount = int32(v)
		case int:
			retryCount = int32(v)
		}
	}

	retryCount++
	if retryCount > 5 {
		slog.Warn("deficit message exceeded max requeues, routing to DLX",
			slog.String("sku", fm.sku),
			slog.Int("qty", fm.qty),
			slog.Int("retries", int(retryCount)),
		)
		// Ack original to remove from pool, then publish to dead letter queue
		_ = conn.Ack(fm.deliveryTag)
		ch := conn.Channel()
		if ch != nil {
			fm.headers["x-retry-count"] = retryCount
			return ch.PublishWithContext(ctx, messaging.DLXExchange, messaging.DLXQueue, false, false, amqp.Publishing{
				ContentType: "application/json",
				Body:        fm.body,
				Headers:     fm.headers,
			})
		}
		return errors.New("channel unavailable")
	}

	fm.headers["x-retry-count"] = retryCount

	// Calculate exponential backoff: 2^retryCount * 50ms (50ms, 100ms, 200ms, 400ms, 800ms)
	backoff := time.Duration(1<<retryCount) * 50 * time.Millisecond
	if backoff > 2*time.Second {
		backoff = 2 * time.Second
	}

	// Ack original immediately to avoid double-consumption during sleep
	_ = conn.Ack(fm.deliveryTag)

	// Publish asynchronously after backoff delay
	go func() {
		time.Sleep(backoff)
		ch := conn.Channel()
		if ch == nil {
			slog.Error("cannot republish backoff message: rabbitmq channel is nil")
			return
		}
		err := ch.PublishWithContext(context.Background(), "", messaging.PoolQueue, false, false, amqp.Publishing{
			ContentType: "application/json",
			Body:        fm.body,
			Headers:     fm.headers,
		})
		if err != nil {
			slog.Error("failed to republish backoff message to pool queue",
				slog.String("sku", fm.sku),
				slog.Any("error", err),
			)
		}
	}()

	return nil
}
