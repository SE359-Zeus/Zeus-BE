package middleware

import (
	"fmt"
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AuditPublisher interface {
	PublishToAudit(msg any) error
}

func Audit(mq AuditPublisher) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Let the handler execute first
		c.Next()

		// Only audit successful modifications (POST, PUT, DELETE)
		method := c.Request.Method
		if method != "POST" && method != "PUT" && method != "DELETE" {
			return
		}

		// Don't audit if the request failed (status >= 400)
		status := c.Writer.Status()
		if status >= 400 {
			return
		}

		// If RabbitMQ is not initialized/running, skip auditing gracefully
		if mq == nil {
			return
		}

		// Extract User ID and Email from context
		var userID uuid.UUID
		if val, exists := c.Get(ContextKeyUserID); exists {
			switch v := val.(type) {
			case uuid.UUID:
				userID = v
			case string:
				if parsed, err := uuid.Parse(v); err == nil {
					userID = parsed
				}
			}
		}

		var userEmail string
		if val, exists := c.Get(ContextKeyEmail); exists {
			userEmail, _ = val.(string)
		}

		if userEmail == "" {
			if authMethod, exists := c.Get(ContextKeyAuthMethod); exists && authMethod == "api_key" {
				if keyName, ok := c.Get("api_key_name"); ok {
					userEmail = fmt.Sprintf("api_key:%v", keyName)
				} else {
					userEmail = "api_key"
				}
			} else {
				userEmail = "system@zeus.local"
			}
		}

		// Map HTTP method to ActionType
		var actionType string
		switch method {
		case "POST":
			actionType = "CREATE"
		case "PUT":
			actionType = "UPDATE"
		case "DELETE":
			actionType = "DELETE"
		}

		// Format details
		var details string
		switch actionType {
		case "CREATE":
			details = fmt.Sprintf("Created resource at %s", c.Request.URL.Path)
		case "UPDATE":
			details = fmt.Sprintf("Updated resource at %s", c.Request.URL.Path)
		case "DELETE":
			details = fmt.Sprintf("Deleted resource at %s", c.Request.URL.Path)
		default:
			details = fmt.Sprintf("%s action at %s", actionType, c.Request.URL.Path)
		}

		// Create the ingest request
		msg := map[string]any{
			"user_id":           userID,
			"user_email":        userEmail,
			"action_type":       actionType,
			"target_resource":   c.Request.URL.Path,
			"details":           details,
			"ip_address":        c.ClientIP(),
			"is_security_event": false,
		}

		// Publish to audit queue asynchronously to avoid blocking the client response
		go func() {
			if err := mq.PublishToAudit(msg); err != nil {
				slog.Warn("audit publish failed",
					slog.String("service", "scm"),
					slog.String("event", "audit_publish_failed"),
					slog.String("method", method),
					slog.String("path", c.Request.URL.Path),
					slog.Any("error", err),
				)
			}
		}()
	}
}
