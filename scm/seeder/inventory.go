package seeder

import (
	"time"

	"zeus-scm-service/internal/models"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func seedInventory(db *gorm.DB, catMap map[string]models.PartCatalog, suppliers []models.Supplier) {
	var existingCount int64
	if err := db.Model(&models.ComponentStock{}).Count(&existingCount).Error; err == nil && existingCount > 0 {
		return
	}

	for _, cat := range catMap {
		sup := suppliers[gofakeit.Number(0, len(suppliers)-1)]
		desc := ""
		if cat.Description != nil {
			desc = *cat.Description
		}
		qty := gofakeit.Number(10, 500)
		stk := models.ComponentStock{
			SKU:               cat.PartNumber,
			Name:              desc,
			Category:          "Components",
			StockQty:          qty,
			ReorderPoint:      gofakeit.Number(5, 20),
			UnitCost:          gofakeit.Float64Range(1.0, 500.0),
			Status:            models.ComponentStatusInStock,
			PrimarySupplierID: sup.ID,
			LeadTimeDays:      sup.LeadTimeDays,
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
				Reference:      "Initial stock load",
				ReferenceType:  models.LedgerRefInitial,
				ReferenceID:    stk.SKU,
				CreatedAt:      time.Now(),
			})
		}
	}
}
