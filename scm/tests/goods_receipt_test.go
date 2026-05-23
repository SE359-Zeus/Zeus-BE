package tests

import (
	"context"
	"testing"

	"zeus-scm-service/internal/models"
	"zeus-scm-service/internal/repository/sqlite"
	"zeus-scm-service/internal/service"

	"github.com/stretchr/testify/assert"
)

func TestGoodsReceipt_ParallelLockProcedure(t *testing.T) {
	db := setupTestDB()
	db.AutoMigrate(&models.GoodsReceipt{}, &models.GRLineItem{}, &models.PurchaseOrder{}, &models.POLineItem{}, &models.ComponentStock{})
	grRepo := sqlite.NewGoodsReceiptRepository(db)
	stockRepo := sqlite.NewStockRepository(db)
	poRepo := sqlite.NewPORepository(db)
	svc := service.NewGoodsReceiptService(db, grRepo, stockRepo, poRepo, 5)

	err := svc.AcquireLock(context.Background(), "GR-2024-301", "Operator-A")
	assert.Error(t, err, "Should fail when GR does not exist")
}

func TestGoodsReceipt_BlindReceiving(t *testing.T) {
	db := setupTestDB()
	db.AutoMigrate(&models.GoodsReceipt{}, &models.GRLineItem{}, &models.PurchaseOrder{}, &models.POLineItem{}, &models.ComponentStock{})
	grRepo := sqlite.NewGoodsReceiptRepository(db)
	stockRepo := sqlite.NewStockRepository(db)
	poRepo := sqlite.NewPORepository(db)
	svc := service.NewGoodsReceiptService(db, grRepo, stockRepo, poRepo, 5)

	counts := map[string]struct {
		Received  int
		Defective int
	}{
		"SOC-XM100-PRO": {Received: 100, Defective: 0},
	}

	err := svc.ProcessBlindReceipt(context.Background(), "GR-2024-301", "Operator-A", counts)
	assert.Error(t, err, "Should fail when GR does not exist")
}

func TestGoodsReceipt_ReleaseLock(t *testing.T) {
	db := setupTestDB()
	db.AutoMigrate(&models.GoodsReceipt{}, &models.GRLineItem{}, &models.PurchaseOrder{}, &models.POLineItem{}, &models.ComponentStock{})
	grRepo := sqlite.NewGoodsReceiptRepository(db)
	stockRepo := sqlite.NewStockRepository(db)
	poRepo := sqlite.NewPORepository(db)
	svc := service.NewGoodsReceiptService(db, grRepo, stockRepo, poRepo, 5)

	err := svc.ReleaseLock(context.Background(), "GR-2024-301")
	assert.Error(t, err, "Should fail when GR does not exist")
}
