package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"zeus-sales-service/config"
	"zeus-sales-service/internal/middlewares"
	"zeus-sales-service/internal/models"
	"zeus-sales-service/internal/repository"

	"github.com/google/uuid"
)

type OrderService struct {
	repo    repository.DbRepository
	cache   repository.CacheRepository
	clients *ClientService
}

func NewOrderService(repo repository.DbRepository, cache repository.CacheRepository, clients *ClientService) *OrderService {
	return &OrderService{repo: repo, cache: cache, clients: clients}
}

func (svc *OrderService) CreateOrder(ctx context.Context, req models.CreateOrderRequest) (*models.OrderResponse, error) {
	if strings.TrimSpace(req.ClientName) == "" {
		return nil, fmt.Errorf("%w: client name is required", middlewares.ErrValidation)
	}
	if req.RequiredDate.IsZero() {
		return nil, fmt.Errorf("%w: required date is required", middlewares.ErrValidation)
	}
	if len(req.Items) == 0 {
		return nil, fmt.Errorf("%w: at least one order item is required", middlewares.ErrValidation)
	}
	seen := make(map[string]struct{}, len(req.Items))
	totalValue := 0.0
	for _, item := range req.Items {
		sku := strings.TrimSpace(item.SKU)
		if sku == "" {
			return nil, fmt.Errorf("%w: sku is required", middlewares.ErrValidation)
		}
		if item.RequestedQty <= 0 {
			return nil, fmt.Errorf("%w: requested quantity for %s must be positive", middlewares.ErrValidation, sku)
		}
		if item.UnitPrice < 0 {
			return nil, fmt.Errorf("%w: unit price for %s cannot be negative", middlewares.ErrValidation, sku)
		}
		key := strings.ToUpper(sku)
		if _, exists := seen[key]; exists {
			return nil, fmt.Errorf("%w: duplicate sku %s", middlewares.ErrValidation, sku)
		}
		seen[key] = struct{}{}
		totalValue += float64(item.RequestedQty) * item.UnitPrice
	}
	client, err := svc.clients.ResolveOrCreateClient(ctx, req.ClientName, req.DestinationAddress, req.ClientTier)
	if err != nil {
		return nil, err
	}
	pendingStatus, err := svc.repo.GetOrderStatusByCode(ctx, models.SalesOrderStatusPendingCode)
	if err != nil {
		return nil, err
	}
	order := &models.SalesOrder{
		ID:                 uuid.New(),
		ClientID:           client.ID,
		ClientName:         client.Name,
		DestinationAddress: strings.TrimSpace(req.DestinationAddress),
		RequiredDate:       req.RequiredDate,
		StatusID:           pendingStatus.ID,
		Status:             pendingStatus,
		TotalValue:         totalValue,
		Locked:             false,
		CreatedAt:          time.Now().UTC(),
	}
	if order.DestinationAddress == "" {
		order.DestinationAddress = client.DefaultDestinationAddress
	}
	if err := svc.repo.CreateOrder(ctx, order); err != nil {
		return nil, err
	}
	items := make([]models.SalesOrderItem, 0, len(req.Items))
	for _, item := range req.Items {
		salesItem := models.SalesOrderItem{
			ID:           uuid.New(),
			OrderID:      order.ID,
			SKU:          strings.TrimSpace(item.SKU),
			RequestedQty: item.RequestedQty,
			AllocatedQty: 0,
			UnitPrice:    item.UnitPrice,
			CreatedAt:    time.Now().UTC(),
		}
		items = append(items, salesItem)
		if err := svc.repo.CreateOrderItem(ctx, &salesItem); err != nil {
			return nil, err
		}
	}
	client.TotalLifetimeOrders++
	if err := svc.clients.repo.UpdateClient(ctx, client); err != nil {
		return nil, err
	}
	if svc.cache != nil {
		if err := svc.cache.EnqueueOrder(ctx, models.AllocationQueueEntry{
			OrderID:      order.ID,
			ClientID:     client.ID,
			ClientTier:   client.Tier,
			RequiredDate: order.RequiredDate,
			IngestedAt:   order.CreatedAt,
		}); err != nil {
			// The order is already persisted; cache warmup must not fail the request.
		}
	}
	return svc.buildResponse(ctx, order, items)
}

func (svc *OrderService) GetOrder(ctx context.Context, id uuid.UUID) (*models.OrderResponse, error) {
	if id == uuid.Nil {
		return nil, fmt.Errorf("%w: order id is required", middlewares.ErrValidation)
	}
	order, err := svc.repo.GetOrder(ctx, id)
	if err != nil {
		return nil, err
	}
	items, err := svc.repo.GetOrderItems(ctx, id)
	if err != nil {
		return nil, err
	}
	return svc.buildResponse(ctx, order, items)
}

func (svc *OrderService) ListOrders(ctx context.Context) ([]models.OrderListItemResponse, error) {
	orders, err := svc.repo.ListOrders(ctx)
	if err != nil {
		return nil, err
	}
	responses := make([]models.OrderListItemResponse, 0, len(orders))
	for _, order := range orders {
		client, _ := svc.repo.GetClient(ctx, order.ClientID)
		if client == nil {
			client = &models.Client{}
		}
		status := ""
		if order.Status != nil {
			status = order.Status.Code
		}
		responses = append(responses, models.OrderListItemResponse{
			OrderID:      order.ID,
			ClientName:   client.Name,
			RequiredDate: order.RequiredDate,
			TotalValue:   order.TotalValue,
			Status:       status,
		})
	}
	return responses, nil
}

func (svc *OrderService) ListOrdersWithFilters(ctx context.Context, states []string, date *time.Time) ([]models.OrderListItemResponse, error) {
	orders, err := svc.repo.ListOrders(ctx)
	if err != nil {
		return nil, err
	}
	// normalize state codes set
	stateSet := map[string]struct{}{}
	for _, s := range states {
		stateSet[strings.ToUpper(strings.TrimSpace(s))] = struct{}{}
	}
	responses := make([]models.OrderListItemResponse, 0, len(orders))
	for _, order := range orders {
		// filter by date if provided (compare date part of RequiredDate)
		if date != nil {
			y1, m1, d1 := order.RequiredDate.Date()
			y2, m2, d2 := date.Date()
			if y1 != y2 || m1 != m2 || d1 != d2 {
				continue
			}
		}
		// load status code for filtering
		includeByState := true
		if len(stateSet) > 0 {
			status, err := svc.repo.GetOrderStatusByID(ctx, order.StatusID)
			if err != nil {
				continue
			}
			if _, ok := stateSet[strings.ToUpper(status.Code)]; !ok {
				includeByState = false
			}
		}
		if !includeByState {
			continue
		}
		client, _ := svc.repo.GetClient(ctx, order.ClientID)
		if client == nil {
			client = &models.Client{}
		}
		status := ""
		if order.Status != nil {
			status = order.Status.Code
		}
		responses = append(responses, models.OrderListItemResponse{
			OrderID:      order.ID,
			ClientName:   client.Name,
			RequiredDate: order.RequiredDate,
			TotalValue:   order.TotalValue,
			Status:       status,
		})
	}
	return responses, nil
}

type MetricsResponse struct {
	TotalPending          int     `json:"total_pending"`
	ActiveProcessingValue float64 `json:"active_processing_value"`
	CompletedLast24Hours  int     `json:"completed_24h"`
}

func (svc *OrderService) GetMetrics(ctx context.Context) (*MetricsResponse, error) {
	orders, err := svc.repo.ListOrders(ctx)
	if err != nil {
		return nil, err
	}
	var totalPending int
	var activeProcessingValue float64
	var completed24 int
	now := time.Now().UTC()
	cutoff := now.Add(-24 * time.Hour)
	for _, order := range orders {
		status, err := svc.repo.GetOrderStatusByID(ctx, order.StatusID)
		if err != nil || status == nil {
			continue
		}
		switch status.Code {
		case models.SalesOrderStatusPendingCode:
			totalPending++
		case models.SalesOrderStatusProcessingCode:
			activeProcessingValue += order.TotalValue
		case models.SalesOrderStatusCompletedCode:
			if order.UpdatedAt.After(cutoff) {
				completed24++
			}
		}
	}
	return &MetricsResponse{TotalPending: totalPending, ActiveProcessingValue: activeProcessingValue, CompletedLast24Hours: completed24}, nil
}

func (svc *OrderService) ListPendingOrders(ctx context.Context) ([]models.OrderListItemResponse, error) {
	orders, err := svc.repo.ListPendingOrders(ctx)
	if err != nil {
		return nil, err
	}
	responses := make([]models.OrderListItemResponse, 0, len(orders))
	for _, order := range orders {
		client, _ := svc.repo.GetClient(ctx, order.ClientID)
		if client == nil {
			client = &models.Client{}
		}
		status := ""
		if order.Status != nil {
			status = order.Status.Code
		}
		responses = append(responses, models.OrderListItemResponse{
			OrderID:      order.ID,
			ClientName:   client.Name,
			RequiredDate: order.RequiredDate,
			TotalValue:   order.TotalValue,
			Status:       status,
		})
	}
	return responses, nil
}

func (svc *OrderService) UpdateOrder(ctx context.Context, id uuid.UUID, req models.UpdateOrderRequest) (*models.OrderResponse, error) {
	if id == uuid.Nil {
		return nil, fmt.Errorf("%w: order id is required", middlewares.ErrValidation)
	}
	order, err := svc.repo.GetOrder(ctx, id)
	if err != nil {
		return nil, err
	}
	status, err := svc.resolveOrderStatus(ctx, order)
	if err != nil {
		return nil, err
	}
	if order.Locked || status == nil || status.Code != models.SalesOrderStatusPendingCode {
		return nil, fmt.Errorf("%w: order is locked or no longer editable", middlewares.ErrConflict)
	}
	if req.DestinationAddress == nil && req.RequiredDate == nil && len(req.Items) == 0 {
		return nil, fmt.Errorf("%w: update request is empty", middlewares.ErrValidation)
	}
	if req.DestinationAddress != nil {
		order.DestinationAddress = strings.TrimSpace(*req.DestinationAddress)
	}
	if req.RequiredDate != nil {
		order.RequiredDate = req.RequiredDate.UTC()
	}
	if len(req.Items) > 0 {
		seen := map[string]struct{}{}
		items := make([]models.SalesOrderItem, 0, len(req.Items))
		total := 0.0
		for _, item := range req.Items {
			sku := strings.TrimSpace(item.SKU)
			if sku == "" {
				return nil, fmt.Errorf("%w: sku is required", middlewares.ErrValidation)
			}
			if item.RequestedQty <= 0 {
				return nil, fmt.Errorf("%w: requested quantity for %s must be positive", middlewares.ErrValidation, sku)
			}
			key := strings.ToUpper(sku)
			if _, exists := seen[key]; exists {
				return nil, fmt.Errorf("%w: duplicate sku %s", middlewares.ErrValidation, sku)
			}
			seen[key] = struct{}{}
			total += float64(item.RequestedQty) * item.UnitPrice
			items = append(items, models.SalesOrderItem{
				ID:           uuid.New(),
				OrderID:      order.ID,
				SKU:          sku,
				RequestedQty: item.RequestedQty,
				UnitPrice:    item.UnitPrice,
				CreatedAt:    time.Now().UTC(),
			})
		}
		order.TotalValue = total
		if err := svc.repo.ReplaceOrderItems(ctx, order.ID, items); err != nil {
			return nil, err
		}
	}
	if err := svc.repo.UpdateOrder(ctx, order); err != nil {
		return nil, err
	}
	if svc.cache != nil {
		if err := svc.cache.ClearQueue(ctx); err != nil {
			return nil, err
		}
	}
	items, _ := svc.repo.GetOrderItems(ctx, order.ID)
	client, _ := svc.repo.GetClient(ctx, order.ClientID)
	if client == nil {
		client = &models.Client{}
	}
	return &models.OrderResponse{Order: *order, Client: *client, Items: items}, nil
}

func (svc *OrderService) CancelOrder(ctx context.Context, id uuid.UUID) error {
	if id == uuid.Nil {
		return fmt.Errorf("%w: order id is required", middlewares.ErrValidation)
	}
	order, err := svc.repo.GetOrder(ctx, id)
	if err != nil {
		return err
	}
	status, err := svc.resolveOrderStatus(ctx, order)
	if err != nil {
		return err
	}
	if order.Locked || status == nil || status.Code != models.SalesOrderStatusPendingCode {
		return fmt.Errorf("%w: order cannot be cancelled once processing has started", middlewares.ErrConflict)
	}
	cancelledStatus, err := svc.repo.GetOrderStatusByCode(ctx, models.SalesOrderStatusCancelledCode)
	if err != nil {
		return err
	}
	order.StatusID = cancelledStatus.ID
	order.Status = cancelledStatus
	order.UpdatedAt = time.Now().UTC()
	if err := svc.repo.UpdateOrder(ctx, order); err != nil {
		return err
	}
	if svc.cache != nil {
		_ = svc.cache.ClearQueue(ctx)
	}
	return nil
}

func (svc *OrderService) resolveOrderStatus(ctx context.Context, order *models.SalesOrder) (*models.SalesOrderStatusLUT, error) {
	if order == nil {
		return nil, nil
	}
	if order.Status != nil && strings.TrimSpace(order.Status.Code) != "" {
		return order.Status, nil
	}
	if order.StatusID == uuid.Nil {
		return nil, nil
	}
	status, err := svc.repo.GetOrderStatusByID(ctx, order.StatusID)
	if err != nil {
		return nil, err
	}
	order.Status = status
	return status, nil
}

// ReserveInventory sends the order items to the MRP service to reserve inventory and trigger MRP processing.
func (svc *OrderService) ReserveInventory(ctx context.Context, id uuid.UUID) error {
	if id == uuid.Nil {
		return fmt.Errorf("%w: order id is required", middlewares.ErrValidation)
	}
	order, err := svc.repo.GetOrder(ctx, id)
	if err != nil {
		return err
	}
	items, err := svc.repo.GetOrderItems(ctx, id)
	if err != nil {
		return err
	}
	// build request payload
	type itemPayload struct {
		SKU string `json:"sku"`
		Qty int    `json:"qty"`
	}
	payload := struct {
		OrderID string        `json:"order_id"`
		Items   []itemPayload `json:"items"`
	}{
		OrderID: order.ID.String(),
		Items:   []itemPayload{},
	}
	for _, it := range items {
		payload.Items = append(payload.Items, itemPayload{SKU: it.SKU, Qty: it.RequestedQty})
	}
	b, _ := json.Marshal(payload)
	mrpURL := config.GetMRPURL()
	endpoint := fmt.Sprintf("%s/api/v1/mrp/reserve", mrpURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("mrp service returned status %d", resp.StatusCode)
	}
	return nil
}

func (svc *OrderService) buildResponse(ctx context.Context, order *models.SalesOrder, items []models.SalesOrderItem) (*models.OrderResponse, error) {
	if order.Status == nil && order.StatusID != uuid.Nil {
		status, err := svc.repo.GetOrderStatusByID(ctx, order.StatusID)
		if err != nil {
			return nil, err
		}
		order.Status = status
	}
	client, err := svc.repo.GetClient(ctx, order.ClientID)
	if err != nil {
		return nil, err
	}
	if client == nil {
		return nil, repository.ErrNotFound
	}
	return &models.OrderResponse{Order: *order, Client: *client, Items: items}, nil
}
