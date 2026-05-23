package service

import (
	"context"
	"fmt"
	"zeus-mrp-service/internal/models"

	"github.com/google/uuid"
)

func (s *ProductionService) GetDemandSummary(ctx context.Context) ([]models.DemandPOSummary, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	orders, err := s.repo.GetOpenProductionOrders(ctx)
	if err != nil {
		return nil, err
	}

	if orders == nil {
		return []models.DemandPOSummary{}, nil
	}

	result := make([]models.DemandPOSummary, 0, len(orders))
	for _, order := range orders {
		shortages, err := s.repo.GetShortagesByOrderID(ctx, order.ID)
		if err != nil {
			return nil, err
		}

		result = append(result, models.DemandPOSummary{
			OrderID:      order.ID.String(),
			TargetBuild:  order.ProductModelCode,
			Quantity:     order.TargetQuantity,
			Status:       string(order.Status),
			MissingCount: len(shortages),
		})
	}

	return result, nil
}

func (s *ProductionService) GeneratePOsForShortages(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	_, err := s.repo.GetAggregatedShortages(ctx)
	if err != nil {
		return err
	}

	// PO publishing/handoff can be plugged here when SCM queue integration is added.
	return nil
}

func (s *ProductionService) GeneratePickList(ctx context.Context, orderID uuid.UUID) (*models.PickListDTO, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if orderID == uuid.Nil {
		return nil, fmt.Errorf("orderID must not be nil")
	}

	order, err := s.repo.GetProductionOrder(ctx, orderID)
	if err != nil {
		return nil, err
	}

	if order == nil {
		return nil, nil
	}

	bomEntries, err := s.repo.GetBOMByModelCode(ctx, order.ProductModelCode)
	if err != nil {
		return nil, err
	}

	parts := make(map[uuid.UUID]*models.PickListItem)
	for _, entry := range bomEntries {
		qty := entry.RequiredQuantityPerUnit * order.TargetQuantity
		if qty <= 0 {
			continue
		}

		if existing, ok := parts[entry.ComponentPartID]; ok {
			existing.Quantity += qty
			continue
		}

		parts[entry.ComponentPartID] = &models.PickListItem{
			PartID:      entry.ComponentPartID,
			SKU:         fmt.Sprintf("PART-%s", entry.ComponentPartID.String()[:8]),
			Quantity:    qty,
			BinLocation: "UNASSIGNED",
		}
	}

	components := make([]models.PickListItem, 0, len(parts))
	for _, item := range parts {
		components = append(components, *item)
	}

	return &models.PickListDTO{
		OrderID:    orderID,
		Components: components,
	}, nil
}

func (s *ProductionService) GetAggregatedDemand(ctx context.Context) ([]models.BOMExplosionResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	aggregated, err := s.repo.GetAggregatedShortages(ctx)
	if err != nil {
		return nil, err
	}

	if aggregated == nil {
		return []models.BOMExplosionResult{}, nil
	}

	return aggregated, nil
}
