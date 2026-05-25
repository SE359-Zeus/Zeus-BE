package service

import "strings"

func normalizeAuditActionType(actionType string) string {
	return strings.ToUpper(strings.TrimSpace(actionType))
}
