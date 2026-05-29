package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
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

		results = append(results, models.BOMExplosionResult{
			PartID:           entry.ComponentPartID,
			ComponentSKU:     sku,
			TotalRequiredQty: requiredQty,
			AvailableQty:     availableQty,
			IsShortage:       availableQty < requiredQty,
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

	metrics, err := s.GetReadinessMetrics(ctx)
	if err != nil {
		return nil, err
	}

	payload := struct {
		Metrics *models.ReadinessMetrics    `json:"metrics"`
		Rows    []models.ReadinessMatrixRow `json:"rows"`
	}{
		Metrics: metrics,
		Rows:    rows,
	}

	return json.MarshalIndent(payload, "", "  ")
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
	return s.RunBOMExplosion(ctx, orderID)
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
	if _, ok := readinessStatusSet[order.Status]; ok && order.Status != "" {
		status = order.Status
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

	hasShortage := false
	hasPartial := false
	for _, result := range results {
		if !result.IsShortage {
			continue
		}
		hasShortage = true
		if result.AvailableQty > 0 {
			hasPartial = true
		}
	}

	if !hasShortage {
		return models.StatusClearToBuild
	}
	if hasPartial {
		return models.StatusPartial
	}
	return models.StatusShortage
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
