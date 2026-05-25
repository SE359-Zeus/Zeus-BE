package seeder

import (
	"context"
	"fmt"
	"log"
	"time"

	"zeus-mrp-service/internal/models"
	"zeus-mrp-service/internal/repository"

	"github.com/google/uuid"
)

func SeedAll(ctx context.Context, repo repository.DbRepository, manifestPath string) error {
	if repo == nil {
		return fmt.Errorf("repository is required")
	}

	log.Println("Starting MRP seeder...")

	manifestData, err := loadManifest(manifestPath)
	if err != nil {
		return fmt.Errorf("load manifest: %w", err)
	}

	openOrders, err := repo.GetOpenProductionOrders(ctx)
	if err != nil {
		return err
	}
	allBOMs, err := repo.GetAllBOMs(ctx)
	if err != nil {
		return err
	}
	if len(openOrders) > 0 || len(allBOMs) > 0 {
		log.Println("MRP seed data already exists, skipping.")
		return nil
	}

	now := time.Now().UTC()
	bomEntries, catalogByID, modelByCode, err := buildBOMEntriesFromManifest(manifestData)
	if err != nil {
		return err
	}
	if err := repo.CreateBOMEntries(ctx, bomEntries); err != nil {
		return fmt.Errorf("seed bom entries: %w", err)
	}

	productionOrders := buildProductionOrdersFromManifest(manifestData, bomEntries, now)
	for i := range productionOrders {
		order := productionOrders[i]
		if err := repo.CreateProductionOrder(ctx, &order); err != nil {
			return fmt.Errorf("seed production order %s: %w", order.ProductModelCode, err)
		}
	}

	shortageLogs := buildShortageLogsFromManifest(productionOrders, bomEntries, catalogByID, modelByCode)
	for i := range shortageLogs {
		entry := shortageLogs[i]
		if err := repo.CreateShortageLog(ctx, &entry); err != nil {
			return fmt.Errorf("seed shortage log %s: %w", entry.ID.String(), err)
		}
	}

	log.Println("MRP seeder finished successfully.")
	return nil
}

func buildBOMEntriesFromManifest(m *manifest) ([]models.BomEntry, map[string]manifestCatalog, map[string]manifestModel, error) {
	if m == nil {
		return nil, nil, nil, fmt.Errorf("manifest is required")
	}
	catalogByID := make(map[string]manifestCatalog, len(m.PartCatalogs))
	for _, catalog := range m.PartCatalogs {
		catalogByID[catalog.ID] = catalog
	}
	modelByCode := make(map[string]manifestModel, len(m.ProductModels))
	entries := make([]models.BomEntry, 0)
	for _, model := range m.ProductModels {
		modelByCode[model.ModelCode] = model
		for _, bom := range model.Bom {
			partID, err := uuid.Parse(bom.PartCatalogID)
			if err != nil {
				return nil, nil, nil, fmt.Errorf("invalid part_catalog_id %q for model %s: %w", bom.PartCatalogID, model.ModelCode, err)
			}
			entries = append(entries, models.BomEntry{
				ParentModelCode:         model.ModelCode,
				ComponentPartID:         partID,
				RequiredQuantityPerUnit: int(bom.Quantity),
			})
		}
	}
	return entries, catalogByID, modelByCode, nil
}

func buildProductionOrdersFromManifest(m *manifest, bomEntries []models.BomEntry, now time.Time) []models.ProductionOrder {
	if m == nil || len(m.ProductModels) == 0 {
		return nil
	}
	limit := 3
	if len(m.ProductModels) < limit {
		limit = len(m.ProductModels)
	}
	orders := make([]models.ProductionOrder, 0, limit)
	statuses := []models.ProductionOrderStatus{
		models.StatusClearToBuild,
		models.StatusPartial,
		models.StatusShortage,
	}
	for i := 0; i < limit; i++ {
		model := m.ProductModels[i]
		orderID := stableUUID("mrp:order:" + model.ModelCode)
		baseQty := 4 + i*2
		if len(model.Bom) > 0 {
			baseQty += len(model.Bom)
		}
		orders = append(orders, models.ProductionOrder{
			ID:               orderID,
			ProductModelCode: model.ModelCode,
			TargetQuantity:   baseQty,
			Status:           statuses[i],
			ScheduledAt:      now.Add(time.Duration(48+i*24) * time.Hour),
			CreatedAt:        now,
		})
	}
	return orders
}

func buildShortageLogsFromManifest(orders []models.ProductionOrder, bomEntries []models.BomEntry, _ map[string]manifestCatalog, modelByCode map[string]manifestModel) []models.ShortageLog {
	if len(orders) == 0 {
		return nil
	}
	byModel := make(map[string][]models.BomEntry)
	for _, entry := range bomEntries {
		byModel[entry.ParentModelCode] = append(byModel[entry.ParentModelCode], entry)
	}
	logs := make([]models.ShortageLog, 0)
	for _, order := range orders {
		if order.Status == models.StatusClearToBuild {
			continue
		}
		bomList := byModel[order.ProductModelCode]
		if len(bomList) == 0 {
			continue
		}
		if model, ok := modelByCode[order.ProductModelCode]; ok && len(model.Bom) == 0 {
			continue
		}
		for idx, bom := range bomList {
			multiplier := 1
			if order.Status == models.StatusShortage {
				multiplier = 2
			}
			shortageQty := bom.RequiredQuantityPerUnit * order.TargetQuantity / 4
			if shortageQty < 1 {
				shortageQty = 1
			}
			shortageQty *= multiplier
			logs = append(logs, models.ShortageLog{
				ID:                 stableUUID(fmt.Sprintf("mrp:shortage:%s:%d", order.ID.String(), idx)),
				ProductionOrderID:  order.ID,
				PartID:             bom.ComponentPartID,
				ShortageQty:        shortageQty,
				ResolutionStatusID: 1,
				ResolutionStatus:   models.ResolutionStatusPlanned,
			})
			if order.Status == models.StatusPartial && idx == 1 {
				break
			}
		}
	}
	return logs
}
