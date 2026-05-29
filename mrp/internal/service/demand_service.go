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

	orders, err := s.repo.GetOpenProductionOrders(ctx)
	if err != nil {
		return nil, err
	}

	if orders == nil {
		return []models.DemandPOSummary{}, nil
	}

	result := make([]models.DemandPOSummary, 0, len(orders))
	for _, order := range orders {
		results, err := s.RunBOMExplosion(ctx, order.ID)
		if err != nil {
			return nil, err
		}

		qtyReady := order.TargetQuantity
		missingCount := 0
		uniqueVendors := make(map[uuid.UUID]struct{})

		if len(results) > 0 {
			minBuild := order.TargetQuantity
			for _, res := range results {
				if res.IsShortage {
					missingCount++
				}
				if res.TotalRequiredQty == 0 {
					continue
				}
				canBuild := (res.AvailableQty * order.TargetQuantity) / res.TotalRequiredQty
				if canBuild < minBuild {
					minBuild = canBuild
				}

				if s.scmClient != nil && order.Status != models.StatusPlanned {
					vendorID, _, err := s.scmClient.GetOptimalSupplier(ctx, res.ComponentSKU)
					if err == nil && vendorID != uuid.Nil {
						uniqueVendors[vendorID] = struct{}{}
					}
				}
			}
			if minBuild < 0 {
				qtyReady = 0
			} else {
				qtyReady = minBuild
			}
		}

		poCount := len(uniqueVendors)
		if order.Status == models.StatusPlanned {
			poCount = 0
		}

		productName := s.resolveAssemblyName(ctx, order.ProductModelCode)

		priority := "NORMAL"
		if order.Status == models.StatusShortage || order.Status == models.StatusPartial {
			priority = "HIGH"
		} else if order.Status == models.StatusPlanned {
			priority = "LOW"
		}

		result = append(result, models.DemandPOSummary{
			OrderID:      order.ID.String(),
			TargetBuild:  order.ProductModelCode,
			ProductName:  productName,
			Quantity:     order.TargetQuantity,
			QtyReady:     qtyReady,
			Status:       string(order.Status),
			Priority:     priority,
			MissingCount: missingCount,
			POCount:      poCount,
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

	orders, err := s.repo.GetOpenProductionOrders(ctx)
	if err != nil {
		return nil, err
	}

	metrics := &models.DemandMetrics{}
	metrics.TotalDemandOrders = len(orders)

	for _, order := range orders {
		metrics.TotalUnitsRequired += order.TargetQuantity

		results, err := s.RunBOMExplosion(ctx, order.ID)
		if err != nil {
			return nil, err
		}
		status := deriveReadinessStatus(results)

		switch status {
		case models.StatusClearToBuild:
			metrics.ReadyToBuild++
		case models.StatusShortage, models.StatusPartial:
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
