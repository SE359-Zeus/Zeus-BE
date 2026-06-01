package service_test

import (
	"context"
	"testing"
	"time"

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
		_, hasLockedBy := fields["locked_by"]
		_, hasExpiresAt := fields["lock_expires_at"]
		return hasLockedBy && hasExpiresAt
	})).Return(nil)

	expiresAt, err := svc.AcquireDispatchLock(context.Background(), "SH-2025-001", "operator-1")
	assert.NoError(t, err)
	assert.NotNil(t, expiresAt)
	assert.True(t, expiresAt.After(time.Now().Add(29*time.Minute)))
	shipmentRepo.AssertExpectations(t)
}

func TestShipmentService_AcquireDispatchLock_NotFound(t *testing.T) {
	svc, shipmentRepo, _ := setupShipmentSvc()

	shipmentRepo.On("GetShipmentByID", anyCtx, "SH-UNKNOWN").Return(nil, assert.AnError)

	expiresAt, err := svc.AcquireDispatchLock(context.Background(), "SH-UNKNOWN", "operator-1")
	assert.ErrorIs(t, err, service.ErrNotFound)
	assert.Nil(t, expiresAt)
	shipmentRepo.AssertExpectations(t)
}

func TestShipmentService_AcquireDispatchLock_AlreadyInTransit(t *testing.T) {
	svc, shipmentRepo, _ := setupShipmentSvc()
	shipment := &models.Shipment{
		ID:     "SH-2025-001",
		Status: models.ShipmentStatusInTransit,
	}

	shipmentRepo.On("GetShipmentByID", anyCtx, "SH-2025-001").Return(shipment, nil)

	expiresAt, err := svc.AcquireDispatchLock(context.Background(), "SH-2025-001", "operator-1")
	assert.ErrorIs(t, err, service.ErrShipmentAlreadyDispatched)
	assert.Nil(t, expiresAt)
	shipmentRepo.AssertExpectations(t)
}

func TestShipmentService_AcquireDispatchLock_AlreadyDelivered(t *testing.T) {
	svc, shipmentRepo, _ := setupShipmentSvc()
	shipment := &models.Shipment{
		ID:     "SH-2025-001",
		Status: models.ShipmentStatusDelivered,
	}

	shipmentRepo.On("GetShipmentByID", anyCtx, "SH-2025-001").Return(shipment, nil)

	expiresAt, err := svc.AcquireDispatchLock(context.Background(), "SH-2025-001", "operator-1")
	assert.ErrorIs(t, err, service.ErrShipmentAlreadyDispatched)
	assert.Nil(t, expiresAt)
	shipmentRepo.AssertExpectations(t)
}

func TestShipmentService_AcquireDispatchLock_LockedByAnotherOperator(t *testing.T) {
	svc, shipmentRepo, _ := setupShipmentSvc()
	otherOperator := "other-operator"
	futureTime := time.Now().Add(10 * time.Minute)
	shipment := &models.Shipment{
		ID:            "SH-2025-001",
		Status:        models.ShipmentStatusScheduled,
		LockedBy:      &otherOperator,
		LockExpiresAt: &futureTime,
	}

	shipmentRepo.On("GetShipmentByID", anyCtx, "SH-2025-001").Return(shipment, nil)

	expiresAt, err := svc.AcquireDispatchLock(context.Background(), "SH-2025-001", "operator-1")
	assert.ErrorIs(t, err, service.ErrShipmentLockConflict)
	assert.Nil(t, expiresAt)
	shipmentRepo.AssertExpectations(t)
}

func TestShipmentService_AcquireDispatchLock_ExpiredLockAllowsReacquire(t *testing.T) {
	svc, shipmentRepo, _ := setupShipmentSvc()
	otherOperator := "other-operator"
	pastTime := time.Now().Add(-10 * time.Minute)
	shipment := &models.Shipment{
		ID:            "SH-2025-001",
		Status:        models.ShipmentStatusScheduled,
		LockedBy:      &otherOperator,
		LockExpiresAt: &pastTime,
	}

	shipmentRepo.On("GetShipmentByID", anyCtx, "SH-2025-001").Return(shipment, nil)
	shipmentRepo.On("UpdateShipmentFields", anyCtx, "SH-2025-001", mock.MatchedBy(func(fields map[string]interface{}) bool {
		_, hasLockedBy := fields["locked_by"]
		_, hasExpiresAt := fields["lock_expires_at"]
		return hasLockedBy && hasExpiresAt
	})).Return(nil)

	expiresAt, err := svc.AcquireDispatchLock(context.Background(), "SH-2025-001", "operator-1")
	assert.NoError(t, err)
	assert.NotNil(t, expiresAt)
	shipmentRepo.AssertExpectations(t)
}

func TestShipmentService_ReleaseDispatchLock_Success(t *testing.T) {
	svc, shipmentRepo, _ := setupShipmentSvc()
	shipment := &models.Shipment{
		ID:     "SH-2025-001",
		Status: models.ShipmentStatusScheduled,
	}

	shipmentRepo.On("GetShipmentByID", anyCtx, "SH-2025-001").Return(shipment, nil)
	shipmentRepo.On("UpdateShipmentFields", anyCtx, "SH-2025-001", mock.MatchedBy(func(fields map[string]interface{}) bool {
		return fields["locked_by"] == nil && fields["lock_expires_at"] == nil
	})).Return(nil)

	err := svc.ReleaseDispatchLock(context.Background(), "SH-2025-001")
	assert.NoError(t, err)
	shipmentRepo.AssertExpectations(t)
}

func TestShipmentService_ReleaseDispatchLock_NotFound(t *testing.T) {
	svc, shipmentRepo, _ := setupShipmentSvc()

	shipmentRepo.On("GetShipmentByID", anyCtx, "SH-UNKNOWN").Return(nil, assert.AnError)

	err := svc.ReleaseDispatchLock(context.Background(), "SH-UNKNOWN")
	assert.ErrorIs(t, err, service.ErrNotFound)
	shipmentRepo.AssertExpectations(t)
}
