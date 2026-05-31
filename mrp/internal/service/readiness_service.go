package service

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"strings"
	"zeus-mrp-service/internal/infrastructure/messaging"
	"zeus-mrp-service/internal/models"

	"github.com/google/uuid"
)

const (
	readinessMatrixCacheKey  = "mrp:readiness:matrix:v1"
	readinessMetricsCacheKey = "mrp:readiness:metrics:v1"
	readinessOrderKeyPrefix  = "mrp:readiness:order:v1:"
	inventoryCacheKeyPrefix  = "mrp:inventory:part:v1:"
)

var readinessStatusSet = map[models.ProductionOrderStatus]struct{}{
	models.StatusPlanned:      {},
	models.StatusClearToBuild: {},
	models.StatusPartial:      {},
	models.StatusShortage:     {},
}

func (s *ProductionService) CheckClearToBuild(ctx context.Context, orderID uuid.UUID) (bool, error) {
	row, err := s.buildReadinessRow(ctx, orderID)
	if err != nil {
		return false, err
	}
	if row == nil {
		return false, nil
	}

	return row.Status == string(models.StatusClearToBuild), nil
}

func (s *ProductionService) RunBOMExplosion(ctx context.Context, orderID uuid.UUID) ([]models.BOMExplosionResult, error) {
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
		return []models.BOMExplosionResult{}, nil
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
	if len(bomEntries) == 0 {
		return []models.BOMExplosionResult{}, nil
	}

	var activePOs []models.PurchaseOrder
	if s.scmClient != nil {
		if pos, err := s.scmClient.ListPOs(ctx, order.ProductModelCode); err == nil {
			activePOs = pos
		}
	}

	results := make([]models.BOMExplosionResult, 0, len(bomEntries))
	for _, entry := range bomEntries {
		requiredQty := entry.RequiredQuantityPerUnit * order.TargetQuantity
		if requiredQty <= 0 {
			return nil, fmt.Errorf("bom entry %d has invalid required quantity", entry.ID)
		}

		var availableQty int
		var sku string
		if s.scmClient != nil {
			part, err := s.scmClient.GetPartCatalogByID(ctx, entry.ComponentPartID)
			if err != nil {
				return nil, err
			}
			if part == nil || strings.TrimSpace(part.SKU) == "" {
				return nil, fmt.Errorf("component part %s not found", entry.ComponentPartID.String())
			}
			availableQty = part.StockQty
			sku = part.SKU
			s.cacheInventory(ctx, entry.ComponentPartID, availableQty)
			if s.cache != nil {
				_ = s.cache.Set(ctx, "mrp:part:sku:v1:"+entry.ComponentPartID.String(), sku)
			}
		} else {
			var err error
			availableQty, err = s.getPartInventory(ctx, entry.ComponentPartID)
			if err != nil {
				return nil, err
			}
			sku, err = s.resolveComponentSKU(ctx, entry.ComponentPartID)
			if err != nil {
				return nil, err
			}
		}

		allocatedQty := 0
		for _, po := range activePOs {
			status := po.Status
			if status == models.POStatusDraft || status == models.POStatusApproved || status == models.POStatusInTransit || status == models.POStatusPartial {
				for _, line := range po.LineItems {
					if line.SKU == sku {
						if status == models.POStatusPartial {
							remaining := line.OrderedQty - line.ReceivedQty
							if remaining > 0 {
								allocatedQty += remaining
							}
						} else {
							allocatedQty += line.OrderedQty
						}
					}
				}
			}
		}

		results = append(results, models.BOMExplosionResult{
			PartID:           entry.ComponentPartID,
			ComponentSKU:     sku,
			TotalRequiredQty: requiredQty,
			AvailableQty:     availableQty,
			AllocatedQty:     allocatedQty,
			IsShortage:       requiredQty > (availableQty + allocatedQty),
		})
	}

	return results, nil
}

func (s *ProductionService) GetReadinessMatrix(ctx context.Context, filter models.ReadinessFilter, page models.PaginationParams) ([]models.ReadinessMatrixRow, int, error) {
	if err := ctx.Err(); err != nil {
		return nil, 0, err
	}

	if page.Page <= 0 {
		return nil, 0, fmt.Errorf("page must be greater than zero")
	}
	if page.PerPage <= 0 {
		return nil, 0, fmt.Errorf("per_page must be greater than zero")
	}

	filter.Status = strings.ToUpper(strings.TrimSpace(filter.Status))
	filter.Search = strings.ToLower(strings.TrimSpace(filter.Search))
	if filter.Status != "" {
		if _, ok := readinessStatusSet[models.ProductionOrderStatus(filter.Status)]; !ok {
			return nil, 0, fmt.Errorf("status must be one of CLEAR_TO_BUILD, PARTIAL, SHORTAGE, PLANNED")
		}
	}

	rows, err := s.loadReadinessRows(ctx)
	if err != nil {
		return nil, 0, err
	}

	filtered := s.filterReadinessRows(rows, filter)
	return paginateReadinessRows(filtered, page), len(filtered), nil
}

func (s *ProductionService) GetReadinessMetrics(ctx context.Context) (*models.ReadinessMetrics, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if cached := s.cachedReadinessMetrics(ctx); cached != nil {
		return cached, nil
	}

	rows, err := s.loadReadinessRows(ctx)
	if err != nil {
		return nil, err
	}

	metrics := buildReadinessMetrics(rows)
	s.cacheReadinessMetrics(ctx, metrics)

	return metrics, nil
}

func (s *ProductionService) ExportReadinessReport(ctx context.Context) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	rows, err := s.loadReadinessRows(ctx)
	if err != nil {
		return nil, err
	}

	buf := &bytes.Buffer{}
	w := csv.NewWriter(buf)
	_ = w.Write([]string{"order_id", "target_build", "quantity", "status", "component_sku", "required_qty", "available_qty", "is_shortage"})
	for _, row := range rows {
		if len(row.DeficitBreakdown) == 0 {
			_ = w.Write([]string{
				row.OrderID.String(),
				row.TargetBuild,
				fmt.Sprintf("%d", row.Quantity),
				row.Status,
				"", "", "", "",
			})
			continue
		}
		for _, d := range row.DeficitBreakdown {
			isShortage := "false"
			if d.IsShortage {
				isShortage = "true"
			}
			_ = w.Write([]string{
				row.OrderID.String(),
				row.TargetBuild,
				fmt.Sprintf("%d", row.Quantity),
				row.Status,
				d.ComponentSKU,
				fmt.Sprintf("%d", d.TotalRequiredQty),
				fmt.Sprintf("%d", d.AvailableQty),
				isShortage,
			})
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (s *ProductionService) GetReadinessByOrderID(ctx context.Context, orderID uuid.UUID) (*models.ReadinessMatrixRow, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if orderID == uuid.Nil {
		return nil, fmt.Errorf("orderID must not be nil")
	}

	if cached := s.cachedReadinessRow(ctx, orderID); cached != nil {
		return cached, nil
	}

	row, err := s.buildReadinessRow(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, nil
	}

	s.cacheReadinessRow(ctx, row)
	return row, nil
}

func (s *ProductionService) GeneratePOForDeficits(ctx context.Context, orderID uuid.UUID) ([]models.BOMExplosionResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	order, err := s.repo.GetProductionOrder(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if order == nil {
		return nil, fmt.Errorf("production order not found")
	}

	results, err := s.RunBOMExplosion(ctx, orderID)
	if err != nil {
		return nil, err
	}

	var shortages []models.BOMExplosionResult
	for _, res := range results {
		if res.IsShortage {
			shortages = append(shortages, res)
		}
	}

	if len(shortages) > 0 {
		if err := s.executeSCMHandoff(ctx, orderID, order.ProductModelCode, shortages); err != nil {
			return nil, err
		}
	}

	return results, nil
}

func (s *ProductionService) loadReadinessRows(ctx context.Context) ([]models.ReadinessMatrixRow, error) {
	if cached := s.cachedReadinessRows(ctx); cached != nil {
		return cached, nil
	}

	orders, err := s.repo.GetOpenProductionOrders(ctx)
	if err != nil {
		return nil, err
	}
	if orders == nil {
		orders = []models.ProductionOrder{}
	}

	rows := make([]models.ReadinessMatrixRow, 0, len(orders))
	for _, order := range orders {
		row, err := s.buildReadinessRow(ctx, order.ID)
		if err != nil {
			return nil, err
		}
		if row == nil {
			continue
		}
		rows = append(rows, *row)
	}

	s.cacheReadinessRows(ctx, rows)
	return rows, nil
}

func (s *ProductionService) buildReadinessRow(ctx context.Context, orderID uuid.UUID) (*models.ReadinessMatrixRow, error) {
	order, err := s.repo.GetProductionOrder(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if order == nil {
		return nil, nil
	}

	results, err := s.RunBOMExplosion(ctx, orderID)
	if err != nil {
		return nil, err
	}

	status := deriveReadinessStatus(results)
	if status != order.Status {
		if err := s.repo.UpdateProductionOrderStatus(ctx, order.ID, status); err != nil {
			return nil, err
		}
		order.Status = status
	}

	if err := s.syncShortagesInDB(ctx, order.ID, results); err != nil {
		return nil, err
	}

	if results == nil {
		results = []models.BOMExplosionResult{}
	}

	return &models.ReadinessMatrixRow{
		OrderID:          order.ID,
		TargetBuild:      order.ProductModelCode,
		Quantity:         order.TargetQuantity,
		Status:           string(status),
		DeficitBreakdown: results,
	}, nil
}

func deriveReadinessStatus(results []models.BOMExplosionResult) models.ProductionOrderStatus {
	if len(results) == 0 {
		return models.StatusClearToBuild
	}

	for _, result := range results {
		if result.IsShortage {
			return models.StatusShortage
		}
	}
	return models.StatusClearToBuild
}

func buildReadinessMetrics(rows []models.ReadinessMatrixRow) *models.ReadinessMetrics {
	totalOpenOrders := len(rows)
	blockedOrders := 0
	shortageParts := make(map[uuid.UUID]struct{})

	for _, row := range rows {
		if row.Status != string(models.StatusClearToBuild) {
			blockedOrders++
		}
		for _, deficit := range row.DeficitBreakdown {
			if deficit.IsShortage {
				shortageParts[deficit.PartID] = struct{}{}
			}
		}
	}

	readinessRate := 0.0
	if totalOpenOrders > 0 {
		readinessRate = float64(totalOpenOrders-blockedOrders) / float64(totalOpenOrders) * 100
	}

	return &models.ReadinessMetrics{
		TotalOpenOrders:      totalOpenOrders,
		ComponentsInShortage: len(shortageParts),
		BlockedOrders:        blockedOrders,
		SupplyReadinessRate:  readinessRate,
	}
}

func paginateReadinessRows(rows []models.ReadinessMatrixRow, page models.PaginationParams) []models.ReadinessMatrixRow {
	if len(rows) == 0 {
		return []models.ReadinessMatrixRow{}
	}

	start := (page.Page - 1) * page.PerPage
	if start >= len(rows) {
		return []models.ReadinessMatrixRow{}
	}

	end := start + page.PerPage
	if end > len(rows) {
		end = len(rows)
	}

	return append([]models.ReadinessMatrixRow{}, rows[start:end]...)
}

func (s *ProductionService) filterReadinessRows(rows []models.ReadinessMatrixRow, filter models.ReadinessFilter) []models.ReadinessMatrixRow {
	if len(rows) == 0 {
		return []models.ReadinessMatrixRow{}
	}

	filtered := make([]models.ReadinessMatrixRow, 0, len(rows))
	for _, row := range rows {
		if filter.Status != "" && row.Status != filter.Status {
			continue
		}
		if filter.Search != "" && !readinessRowMatchesSearch(row, filter.Search) {
			continue
		}
		filtered = append(filtered, row)
	}

	return filtered
}

func readinessRowMatchesSearch(row models.ReadinessMatrixRow, search string) bool {
	lowerSearch := strings.ToLower(search)
	if strings.Contains(strings.ToLower(row.OrderID.String()), lowerSearch) {
		return true
	}
	if strings.Contains(strings.ToLower(row.TargetBuild), lowerSearch) {
		return true
	}
	if strings.Contains(strings.ToLower(row.Status), lowerSearch) {
		return true
	}
	for _, deficit := range row.DeficitBreakdown {
		if strings.Contains(strings.ToLower(deficit.PartID.String()), lowerSearch) {
			return true
		}
	}
	return false
}

func (s *ProductionService) getPartInventory(ctx context.Context, partID uuid.UUID) (int, error) {
	if partID == uuid.Nil {
		return 0, fmt.Errorf("partID must not be nil")
	}

	if cached := s.cachedInventory(ctx, partID); cached != nil {
		return *cached, nil
	}

	var availableQty int
	if s.scmClient != nil {
		part, err := s.scmClient.GetPartCatalogByID(ctx, partID)
		if err != nil {
			return 0, err
		}
		if part != nil {
			availableQty = part.StockQty
		}
	}

	s.cacheInventory(ctx, partID, availableQty)
	return availableQty, nil
}

func (s *ProductionService) cachedReadinessRows(ctx context.Context) []models.ReadinessMatrixRow {
	if s.cache == nil {
		return nil
	}

	var rows []models.ReadinessMatrixRow
	if err := s.cache.Get(ctx, readinessMatrixCacheKey, &rows); err != nil {
		return nil
	}
	if rows == nil {
		return []models.ReadinessMatrixRow{}
	}

	return rows
}

func (s *ProductionService) cacheReadinessRows(ctx context.Context, rows []models.ReadinessMatrixRow) {
	if s.cache == nil {
		return
	}
	_ = s.cache.Set(ctx, readinessMatrixCacheKey, rows)
}

func (s *ProductionService) cachedReadinessMetrics(ctx context.Context) *models.ReadinessMetrics {
	if s.cache == nil {
		return nil
	}

	var metrics models.ReadinessMetrics
	if err := s.cache.Get(ctx, readinessMetricsCacheKey, &metrics); err != nil {
		return nil
	}
	return &metrics
}

func (s *ProductionService) cacheReadinessMetrics(ctx context.Context, metrics *models.ReadinessMetrics) {
	if s.cache == nil || metrics == nil {
		return
	}
	_ = s.cache.Set(ctx, readinessMetricsCacheKey, metrics)
}

func (s *ProductionService) cachedReadinessRow(ctx context.Context, orderID uuid.UUID) *models.ReadinessMatrixRow {
	if s.cache == nil {
		return nil
	}

	var row models.ReadinessMatrixRow
	if err := s.cache.Get(ctx, readinessOrderKeyPrefix+orderID.String(), &row); err != nil {
		return nil
	}
	if row.OrderID == uuid.Nil {
		return nil
	}

	return &row
}

func (s *ProductionService) cacheReadinessRow(ctx context.Context, row *models.ReadinessMatrixRow) {
	if s.cache == nil || row == nil {
		return
	}
	_ = s.cache.Set(ctx, readinessOrderKeyPrefix+row.OrderID.String(), row)
}

func (s *ProductionService) cachedInventory(ctx context.Context, partID uuid.UUID) *int {
	if s.cache == nil {
		return nil
	}

	var availableQty int
	if err := s.cache.Get(ctx, inventoryCacheKeyPrefix+partID.String(), &availableQty); err != nil {
		return nil
	}
	return &availableQty
}

func (s *ProductionService) cacheInventory(ctx context.Context, partID uuid.UUID, availableQty int) {
	if s.cache == nil {
		return
	}
	_ = s.cache.Set(ctx, inventoryCacheKeyPrefix+partID.String(), availableQty)
}

func (s *ProductionService) syncShortagesInDB(ctx context.Context, orderID uuid.UUID, results []models.BOMExplosionResult) error {
	existingLogs, err := s.repo.GetShortagesByOrderID(ctx, orderID)
	if err != nil {
		return err
	}

	existingMap := make(map[uuid.UUID]models.ShortageLog)
	for _, log := range existingLogs {
		existingMap[log.PartID] = log
	}

	for _, res := range results {
		if res.IsShortage {
			shortageQty := res.TotalRequiredQty - (res.AvailableQty + res.AllocatedQty)
			if shortageQty <= 0 {
				if _, exists := existingMap[res.PartID]; exists {
					if err := s.repo.DeleteShortageLog(ctx, orderID, res.PartID); err != nil {
						return err
					}
				}
				continue
			}

			var partSKU string
			if s.scmClient != nil {
				part, err := s.scmClient.GetPartCatalogByID(ctx, res.PartID)
				if err == nil && part != nil {
					partSKU = part.SKU
				}
			}
			if partSKU == "" {
				partSKU = res.ComponentSKU
			}
			if partSKU == "" {
				partSKU = fmt.Sprintf("PART-%s", res.PartID.String()[:8])
			}

			if existing, exists := existingMap[res.PartID]; exists {
				if existing.ShortageQty != shortageQty {
					existing.ShortageQty = shortageQty
					existing.ResolutionStatus = models.ResolutionStatusShortage
					if err := s.repo.UpdateShortageLog(ctx, &existing); err != nil {
						return err
					}
				}
			} else {
				newLog := &models.ShortageLog{
					ID:                uuid.New(),
					ProductionOrderID: orderID,
					PartID:            res.PartID,
					ShortageQty:       shortageQty,
					ResolutionStatus:  models.ResolutionStatusShortage,
				}
				if err := s.repo.CreateShortageLog(ctx, newLog); err != nil {
					return err
				}

				if s.audit != nil {
					_ = s.audit.PublishJSON(ctx, messaging.DeficitPoolQueue, map[string]any{
						"sku":      partSKU,
						"qty":      shortageQty,
						"order_id": orderID.String(),
					})
				}
			}
		} else {
			if _, exists := existingMap[res.PartID]; exists {
				if err := s.repo.DeleteShortageLog(ctx, orderID, res.PartID); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (s *ProductionService) InvalidateReadinessCache(ctx context.Context, orderID uuid.UUID) {
	if s.cache == nil {
		return
	}
	_ = s.cache.Del(ctx, readinessMatrixCacheKey, readinessMetricsCacheKey)
	if orderID != uuid.Nil {
		_ = s.cache.Del(ctx, readinessOrderKeyPrefix+orderID.String())
	}
}
