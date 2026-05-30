package seeder

import (
	"math/rand"
	"time"

	"zeus-scm-service/internal/models"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func deriveComponentStatus(stockQty, reorderPoint int) models.ComponentStatus {
	switch {
	case stockQty <= 0:
		return models.ComponentStatusOutOfStock
	case stockQty <= reorderPoint:
		return models.ComponentStatusLowStock
	default:
		return models.ComponentStatusInStock
	}
}

func chooseSupplierForSKU(db *gorm.DB, sku string, fallback []models.Supplier) *models.Supplier {
	var mappings []models.SkuMapping
	if err := db.Where("sku = ?", sku).Find(&mappings).Error; err == nil && len(mappings) > 0 {
		selected := mappings[rand.New(rand.NewSource(time.Now().UnixNano())).Intn(len(mappings))]
		for _, supplier := range fallback {
			if supplier.ID == selected.SupplierID {
				return &supplier
			}
		}
		var supplier models.Supplier
		if err := db.First(&supplier, "id = ?", selected.SupplierID).Error; err == nil {
			return &supplier
		}
	}
	if len(fallback) == 0 {
		return nil
	}
	supplier := fallback[rand.New(rand.NewSource(time.Now().UnixNano())).Intn(len(fallback))]
	return &supplier
}

func seedInventory(db *gorm.DB, catMap map[string]models.PartCatalog, suppliers []models.Supplier) {
	var existingCount int64
	if err := db.Model(&models.ComponentStock{}).Count(&existingCount).Error; err == nil && existingCount > 0 {
		return
	}

	for _, cat := range catMap {
		sup := chooseSupplierForSKU(db, cat.PartNumber, suppliers)
		desc := ""
		if cat.Description != nil {
			desc = *cat.Description
		}
		qty := gofakeit.Number(10, 500)
		reorderPoint := gofakeit.Number(5, 20)
		status := deriveComponentStatus(qty, reorderPoint)
		stk := models.ComponentStock{
			SKU:          cat.PartNumber,
			Name:         desc,
			Category:     "Components",
			StockQty:     qty,
			ReorderPoint: reorderPoint,
			UnitCost:     gofakeit.Float64Range(1.0, 500.0),
			Status:       status,
		}
		if sup != nil {
			stk.PrimarySupplierID = sup.ID
			stk.LeadTimeDays = sup.LeadTimeDays
		}
		db.Where("sku = ?", stk.SKU).Assign(stk).FirstOrCreate(&stk)

		var ledgerCount int64
		db.Model(&models.InventoryLedger{}).Where("sku = ? AND reference_type = ?", stk.SKU, models.LedgerRefInitial).Count(&ledgerCount)
		if ledgerCount == 0 {
			db.Create(&models.InventoryLedger{
				ID:             uuid.NewString(),
				SKU:            stk.SKU,
				Type:           models.LedgerTxnTypeIN,
				QtyChange:      qty,
				RunningBalance: qty,
				Location:       "WH-A",
				OperatorID:     "system",
				OperatorName:   "System",
				Reference:      "Initial stock load",
				ReferenceType:  models.LedgerRefInitial,
				ReferenceID:    stk.SKU,
				CreatedAt:      time.Now(),
			})
		}
	}
}
