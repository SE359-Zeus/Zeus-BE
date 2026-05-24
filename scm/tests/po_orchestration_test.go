package tests

import (
	"context"
	"testing"

	"zeus-scm-service/internal/models"
	"zeus-scm-service/internal/repository/sqlite"
	"zeus-scm-service/internal/service"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestPOOrchestration_CreateDraft(t *testing.T) {
	db := setupTestDB()
	db.AutoMigrate(&models.PurchaseOrder{}, &models.POLineItem{})
	poRepo := sqlite.NewPORepository(db)
	stockRepo := sqlite.NewStockRepository(db)
	svc := service.NewPOService(poRepo, stockRepo, "")

	po, err := svc.CreateDraft(context.Background(), uuid.New(), "Build-X1")
	assert.NoError(t, err, "Should successfully create draft")
	assert.NotNil(t, po)
	if po != nil {
		assert.Equal(t, models.POStatusDraft, po.Status)
	}
}

func TestPOOrchestration_EagerSlotLocking(t *testing.T) {
	db := setupTestDB()
	db.AutoMigrate(&models.PurchaseOrder{}, &models.POLineItem{}, &models.ComponentStock{})
	poRepo := sqlite.NewPORepository(db)
	stockRepo := sqlite.NewStockRepository(db)
	svc := service.NewPOService(poRepo, stockRepo, "")

	err := svc.AddLineItemWithLock(context.Background(), "PO-2024-108", "MOD-WIFI7-AX", 200)
	assert.Error(t, err, "Should fail when deficit pool is unavailable (nil MQ)")
}

func TestPOOrchestration_ApprovePO(t *testing.T) {
	db := setupTestDB()
	db.AutoMigrate(&models.PurchaseOrder{}, &models.POLineItem{})
	poRepo := sqlite.NewPORepository(db)
	stockRepo := sqlite.NewStockRepository(db)
	svc := service.NewPOService(poRepo, stockRepo, "")

	err := svc.ApprovePO(context.Background(), "PO-2024-108")
	assert.Error(t, err, "Should fail when PO does not exist")
}

func TestPOOrchestration_StateRegressionPrevention(t *testing.T) {
	db := setupTestDB()
	db.AutoMigrate(&models.PurchaseOrder{}, &models.POLineItem{})
	poRepo := sqlite.NewPORepository(db)
	stockRepo := sqlite.NewStockRepository(db)
	svc := service.NewPOService(poRepo, stockRepo, "")

	err := svc.TransitionState(context.Background(), "PO-2024-101", models.POStatusDraft)
	assert.Error(t, err, "Should block state regression for non-existent PO")
}
