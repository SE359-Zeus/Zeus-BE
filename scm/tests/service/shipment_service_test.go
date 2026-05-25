package service_test

import (
	"context"
	"testing"

	"zeus-scm-service/internal/models"
	"zeus-scm-service/internal/repository"
	"zeus-scm-service/internal/service"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

func setupShipmentSvc() (service.IShipmentService, *repository.MockShipmentRepository, *repository.MockStockRepository) {
	shipmentRepo := new(repository.MockShipmentRepository)
	stockRepo := new(repository.MockStockRepository)
	svc := service.NewShipmentService(&gorm.DB{}, shipmentRepo, stockRepo)
	return svc, shipmentRepo, stockRepo
}

func TestShipmentService_AcquireDispatchLock_Success(t *testing.T) {
	svc, shipmentRepo, _ := setupShipmentSvc()
	shipment := &models.Shipment{
		ID:     "SH-2025-001",
		Status: models.ShipmentStatusScheduled,
	}

	shipmentRepo.On("GetShipmentByID", anyCtx, "SH-2025-001").Return(shipment, nil)
	shipmentRepo.On("UpdateShipmentFields", anyCtx, "SH-2025-001", mock.MatchedBy(func(fields map[string]interface{}) bool {
		_, ok := fields["ship_date"]
		return ok
	})).Return(nil)

	err := svc.AcquireDispatchLock(context.Background(), "SH-2025-001", "operator-1")
	assert.NoError(t, err)
	shipmentRepo.AssertExpectations(t)
}

func TestShipmentService_AcquireDispatchLock_NotFound(t *testing.T) {
	svc, shipmentRepo, _ := setupShipmentSvc()

	shipmentRepo.On("GetShipmentByID", anyCtx, "SH-UNKNOWN").Return(nil, assert.AnError)

	err := svc.AcquireDispatchLock(context.Background(), "SH-UNKNOWN", "operator-1")
	assert.ErrorIs(t, err, service.ErrNotFound)
	shipmentRepo.AssertExpectations(t)
}

func TestShipmentService_AcquireDispatchLock_AlreadyInTransit(t *testing.T) {
	svc, shipmentRepo, _ := setupShipmentSvc()
	shipment := &models.Shipment{
		ID:     "SH-2025-001",
		Status: models.ShipmentStatusInTransit,
	}

	shipmentRepo.On("GetShipmentByID", anyCtx, "SH-2025-001").Return(shipment, nil)

	err := svc.AcquireDispatchLock(context.Background(), "SH-2025-001", "operator-1")
	assert.ErrorIs(t, err, service.ErrAlreadyLocked)
	shipmentRepo.AssertExpectations(t)
}

func TestShipmentService_AcquireDispatchLock_AlreadyDelivered(t *testing.T) {
	svc, shipmentRepo, _ := setupShipmentSvc()
	shipment := &models.Shipment{
		ID:     "SH-2025-001",
		Status: models.ShipmentStatusDelivered,
	}

	shipmentRepo.On("GetShipmentByID", anyCtx, "SH-2025-001").Return(shipment, nil)

	err := svc.AcquireDispatchLock(context.Background(), "SH-2025-001", "operator-1")
	assert.ErrorIs(t, err, service.ErrAlreadyLocked)
	shipmentRepo.AssertExpectations(t)
}
