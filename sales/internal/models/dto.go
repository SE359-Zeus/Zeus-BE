package models

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type OrderItemRequest struct {
	SKU          string  `json:"sku"`
	RequestedQty int     `json:"requestedQty"`
	UnitPrice    float64 `json:"unitPrice"`
}

type CreateOrderRequest struct {
	ClientName         string             `json:"clientName"`
	DestinationAddress string             `json:"destinationAddress"`
	ClientTier         ClientTier         `json:"clientTier"`
	RequiredDate       time.Time          `json:"requiredDate"`
	Items              []OrderItemRequest `json:"items"`
}

type UpdateOrderRequest struct {
	DestinationAddress *string            `json:"destinationAddress"`
	RequiredDate       *time.Time         `json:"requiredDate"`
	Items              []OrderItemRequest `json:"items"`
}

type CreateClientRequest struct {
	Name                      string     `json:"name"`
	Tier                      ClientTier `json:"tier"`
	DefaultDestinationAddress string     `json:"defaultDestinationAddress"`
}

type UpdateClientRequest struct {
	Name                      *string     `json:"name"`
	Tier                      *ClientTier `json:"tier"`
	DefaultDestinationAddress *string     `json:"defaultDestinationAddress"`
}

type OrderResponse struct {
	Order  SalesOrder       `json:"order"`
	Client Client           `json:"client"`
	Items  []SalesOrderItem `json:"items"`
}

type OrderListItemResponse struct {
	OrderID      uuid.UUID `json:"orderId"`
	ClientName   string    `json:"clientName"`
	RequiredDate time.Time `json:"requiredDate"`
	TotalValue   float64   `json:"totalValue"`
	Status       string    `json:"status"`
}

type ClientResponse struct {
	Client Client `json:"client"`
}

type QueueStatus struct {
	Entries []AllocationQueueEntry `json:"entries"`
}

type DispatchNotification struct {
	OrderID uuid.UUID `json:"order_id"`
	Status  string    `json:"status"`
}

type createOrderRequestAlias struct {
	ClientName         string             `json:"clientName"`
	DestinationAddress string             `json:"destinationAddress"`
	ClientTier         ClientTier         `json:"clientTier"`
	RequiredDate       string             `json:"requiredDate"`
	Items              []OrderItemRequest `json:"items"`
}

type updateOrderRequestAlias struct {
	DestinationAddress *string            `json:"destinationAddress"`
	RequiredDate       *string            `json:"requiredDate"`
	Items              []OrderItemRequest `json:"items"`
}

func (req *CreateOrderRequest) UnmarshalJSON(data []byte) error {
	var payload createOrderRequestAlias
	if err := json.Unmarshal(data, &payload); err != nil {
		return err
	}
	requiredDate, err := parseFlexibleTime(payload.RequiredDate)
	if err != nil {
		return fmt.Errorf("invalid requiredDate: %w", err)
	}
	*req = CreateOrderRequest{
		ClientName:         strings.TrimSpace(payload.ClientName),
		DestinationAddress: strings.TrimSpace(payload.DestinationAddress),
		ClientTier:         payload.ClientTier,
		RequiredDate:       requiredDate,
		Items:              payload.Items,
	}
	return nil
}

func (req *UpdateOrderRequest) UnmarshalJSON(data []byte) error {
	var payload updateOrderRequestAlias
	if err := json.Unmarshal(data, &payload); err != nil {
		return err
	}
	var requiredDate *time.Time
	if payload.RequiredDate != nil {
		parsed, err := parseFlexibleTime(*payload.RequiredDate)
		if err != nil {
			return fmt.Errorf("invalid requiredDate: %w", err)
		}
		requiredDate = &parsed
	}
	*req = UpdateOrderRequest{
		DestinationAddress: payload.DestinationAddress,
		RequiredDate:       requiredDate,
		Items:              payload.Items,
	}
	return nil
}

func parseFlexibleTime(value string) (time.Time, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return time.Time{}, fmt.Errorf("requiredDate is required")
	}
	layouts := []struct {
		layout string
		inUTC  bool
	}{
		{time.RFC3339Nano, false},
		{time.RFC3339, false},
		{"2006-1-2T15:04:05", true},
		{"2006-01-02T15:04:05", true},
	}
	for _, candidate := range layouts {
		var parsed time.Time
		var err error
		if candidate.inUTC {
			parsed, err = time.ParseInLocation(candidate.layout, trimmed, time.UTC)
		} else {
			parsed, err = time.Parse(candidate.layout, trimmed)
		}
		if err == nil {
			return parsed.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("unsupported time format %q", value)
}
