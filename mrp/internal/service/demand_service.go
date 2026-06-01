package service

import (
	"context"
	"fmt"
	"zeus-mrp-service/internal/infrastructure/messaging"
	"zeus-mrp-service/internal/models"

	"github.com/google/uuid"
)

func (s *ProductionService) GetDemandSummary(ctx context.Context) ([]models.DemandPOSummary, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	rows, err := s.loadReadinessRows(ctx)
	if err != nil {
		return nil, err
	}

	// Batch-fetch production orders to avoid N+1 queries
	orderMap := make(map[uuid.UUID]*models.ProductionOrder, len(rows))
	for _, row := range rows {
		if _, exists := orderMap[row.OrderID]; !exists {
			order, err := s.repo.GetProductionOrder(ctx, row.OrderID)
			if err == nil && order != nil {
				orderMap[row.OrderID] = order
			}
		}
	}

	// Cache assembly name lookups across rows
	nameCache := make(map[string]string)

	result := make([]models.DemandPOSummary, 0, len(rows))
	for _, row := range rows {
		order := orderMap[row.OrderID]
		if order == nil {
			continue
		}

		qtyReady := row.Quantity
		missingCount := 0

		if len(row.DeficitBreakdown) > 0 {
			minBuild := row.Quantity
			for _, res := range row.DeficitBreakdown {
				if res.IsShortage {
					missingCount++
				}
				if res.TotalRequiredQty == 0 {
					continue
				}
				canBuild := (res.AvailableQty * row.Quantity) / res.TotalRequiredQty
				if canBuild < minBuild {
					minBuild = canBuild
				}
			}
			if minBuild < 0 {
				qtyReady = 0
			} else {
				qtyReady = minBuild
			}
		}

		productName := order.ProductModelCode
		if cached, ok := nameCache[order.ProductModelCode]; ok {
			productName = cached
		} else {
			resolved := s.resolveAssemblyName(ctx, order.ProductModelCode)
			if resolved != order.ProductModelCode {
				nameCache[order.ProductModelCode] = resolved
				productName = resolved
			}
		}

		priority := "NORMAL"
		if row.Status == string(models.StatusShortage) || row.Status == string(models.StatusPartial) {
			priority = "HIGH"
		} else if row.Status == string(models.StatusPlanned) {
			priority = "LOW"
		}

		result = append(result, models.DemandPOSummary{
			OrderID:      row.OrderID.String(),
			TargetBuild:  order.ProductModelCode,
			ProductName:  productName,
			Quantity:     row.Quantity,
			QtyReady:     qtyReady,
			Status:       row.Status,
			Priority:     priority,
			MissingCount: missingCount,
			TargetDate:   order.ScheduledAt,
		})
	}

	return result, nil
}

// GetDemandMetrics returns the KPI card values for the Demand view header.
func (s *ProductionService) GetDemandMetrics(ctx context.Context) (*models.DemandMetrics, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// Use cached readiness rows instead of re-running BOM explosion.
	rows, err := s.loadReadinessRows(ctx)
	if err != nil {
		return nil, err
	}

	metrics := &models.DemandMetrics{}
	metrics.TotalDemandOrders = len(rows)

	for _, row := range rows {
		metrics.TotalUnitsRequired += row.Quantity

		switch row.Status {
		case string(models.StatusClearToBuild):
			metrics.ReadyToBuild++
		case string(models.StatusShortage), string(models.StatusPartial):
			metrics.ShortageOrPartial++
		}
	}

	return metrics, nil
}

func (s *ProductionService) executeSCMHandoff(ctx context.Context, orderID uuid.UUID, targetBuild string, shortages []models.BOMExplosionResult) error {
	if s.scmClient == nil || s.audit == nil {
		return nil
	}

	vendorShortages := make(map[uuid.UUID][]models.BOMExplosionResult)
	for _, res := range shortages {
		vendorID, _, err := s.scmClient.GetOptimalSupplier(ctx, res.ComponentSKU)
		if err != nil {
			return fmt.Errorf("failed to find optimal supplier for SKU %s: %w", res.ComponentSKU, err)
		}
		vendorShortages[vendorID] = append(vendorShortages[vendorID], res)
	}

	for vendorID, items := range vendorShortages {
		poID, err := s.scmClient.CreateDraftPO(ctx, vendorID, targetBuild)
		if err != nil {
			return fmt.Errorf("failed to create draft PO for vendor %s: %w", vendorID, err)
		}

		for _, item := range items {
			shortageQty := item.TotalRequiredQty - item.AvailableQty
			if shortageQty <= 0 {
				continue
			}

			deficitPayload := map[string]any{
				"sku":      item.ComponentSKU,
				"qty":      shortageQty,
				"order_id": orderID.String(),
			}
			if err := s.audit.PublishJSON(ctx, messaging.DeficitPoolQueue, deficitPayload); err != nil {
				return fmt.Errorf("failed to publish deficit to queue for SKU %s: %w", item.ComponentSKU, err)
			}

			if err := s.scmClient.AddLineItemWithLock(ctx, poID, item.ComponentSKU, shortageQty); err != nil {
				return fmt.Errorf("failed to add line item with lock for PO %s, SKU %s: %w", poID, item.ComponentSKU, err)
			}
		}
	}

	return nil
}

func (s *ProductionService) GeneratePOsForShortages(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	orders, err := s.repo.GetOpenProductionOrders(ctx)
	if err != nil {
		return err
	}

	for _, order := range orders {
		results, err := s.RunBOMExplosion(ctx, order.ID)
		if err != nil {
			return err
		}

		var shortages []models.BOMExplosionResult
		for _, res := range results {
			if res.IsShortage {
				shortages = append(shortages, res)
			}
		}

		if len(shortages) > 0 {
			if err := s.executeSCMHandoff(ctx, order.ID, order.ProductModelCode, shortages); err != nil {
				return err
			}
		}
	}

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

	var bomEntries []models.BomEntry
	if s.cache != nil {
		bomEntries, err = s.cache.GetBOMByModelCode(ctx, order.ProductModelCode, s.repo.GetBOMByModelCode)
	} else {
		bomEntries, err = s.repo.GetBOMByModelCode(ctx, order.ProductModelCode)
	}
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

	orders, err := s.repo.GetOpenProductionOrders(ctx)
	if err != nil {
		return nil, err
	}

	aggregated := make(map[uuid.UUID]*models.BOMExplosionResult)
	var orderedPartIDs []uuid.UUID

	for _, order := range orders {
		results, err := s.RunBOMExplosion(ctx, order.ID)
		if err != nil {
			return nil, err
		}

		for _, res := range results {
			if !res.IsShortage {
				continue
			}

			shortageQty := res.TotalRequiredQty - res.AvailableQty
			if shortageQty <= 0 {
				continue
			}

			if existing, exists := aggregated[res.PartID]; exists {
				existing.TotalRequiredQty += shortageQty
			} else {
				orderedPartIDs = append(orderedPartIDs, res.PartID)
				aggregated[res.PartID] = &models.BOMExplosionResult{
					PartID:           res.PartID,
					ComponentSKU:     res.ComponentSKU,
					TotalRequiredQty: shortageQty,
					AvailableQty:     res.AvailableQty,
					IsShortage:       true,
				}
			}
		}
	}

	result := make([]models.BOMExplosionResult, 0, len(orderedPartIDs))
	for _, partID := range orderedPartIDs {
		result = append(result, *aggregated[partID])
	}

	return result, nil
}

func (s *ProductionService) DeleteProductionOrder(ctx context.Context, orderID uuid.UUID) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if orderID == uuid.Nil {
		return fmt.Errorf("orderID must not be nil")
	}

	order, err := s.repo.GetProductionOrder(ctx, orderID)
	if err != nil {
		return err
	}
	if order == nil {
		return fmt.Errorf("production order not found")
	}

	if err := s.repo.DeleteProductionOrder(ctx, orderID); err != nil {
		return fmt.Errorf("failed to delete production order: %w", err)
	}

	s.InvalidateReadinessCache(ctx, orderID)
	return nil
}
