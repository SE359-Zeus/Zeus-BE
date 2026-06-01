package tests

import (
	"context"
	"testing"

	"zeus-scm-service/internal/models"
	"zeus-scm-service/internal/repository/sqlite"
	"zeus-scm-service/internal/service"

	"github.com/stretchr/testify/assert"
)

func TestShipment_DispatchLockingProcedure(t *testing.T) {
	db := setupTestDB()
	db.AutoMigrate(&models.Shipment{}, &models.ShipmentItem{})
	shipmentRepo := sqlite.NewShipmentRepository(db)
	stockRepo := sqlite.NewStockRepository(db)
	svc := service.NewShipmentService(db, shipmentRepo, stockRepo)

	expiresAt, err := svc.AcquireDispatchLock(context.Background(), "SHP-2024-201", "Operator-B")
	assert.Error(t, err, "Should fail when shipment does not exist")
	assert.Nil(t, expiresAt)
}

func TestShipment_InventoryDeductionTrigger(t *testing.T) {
	db := setupTestDB()
	db.AutoMigrate(&models.Shipment{}, &models.ShipmentItem{}, &models.ComponentStock{})
	shipmentRepo := sqlite.NewShipmentRepository(db)
	stockRepo := sqlite.NewStockRepository(db)
	svc := service.NewShipmentService(db, shipmentRepo, stockRepo)

	err := svc.DispatchShipment(context.Background(), "SHP-2024-201", "Operator-B")
	assert.Error(t, err, "Should fail when shipment does not exist")
}
