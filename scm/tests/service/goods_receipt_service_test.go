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

func setupGRSvc() (service.IGoodsReceiptService, *repository.MockGoodsReceiptRepository, *repository.MockStockRepository, *repository.MockPORepository) {
	grRepo := new(repository.MockGoodsReceiptRepository)
	stockRepo := new(repository.MockStockRepository)
	poRepo := new(repository.MockPORepository)
	svc := service.NewGoodsReceiptService(&gorm.DB{}, grRepo, stockRepo, poRepo, 5)
	return svc, grRepo, stockRepo, poRepo
}

func TestGoodsReceiptService_AcquireLock_Success(t *testing.T) {
	svc, grRepo, _, _ := setupGRSvc()
	gr := &models.GoodsReceipt{
		ID:     "GR-2025-001",
		Status: models.GRStatusPending,
	}

	grRepo.On("GetGRByID", anyCtx, "GR-2025-001").Return(gr, nil)
	grRepo.On("UpdateGRFields", anyCtx, "GR-2025-001", mock.MatchedBy(func(fields map[string]interface{}) bool {
		_, hasLockedBy := fields["locked_by"]
		_, hasExpiresAt := fields["lock_expires_at"]
		return hasLockedBy && hasExpiresAt
	})).Return(nil)

	err := svc.AcquireLock(context.Background(), "GR-2025-001", "operator-1")
	assert.NoError(t, err)
	grRepo.AssertExpectations(t)
}

func TestGoodsReceiptService_AcquireLock_NotFound(t *testing.T) {
	svc, grRepo, _, _ := setupGRSvc()

	grRepo.On("GetGRByID", anyCtx, "GR-UNKNOWN").Return(nil, assert.AnError)

	err := svc.AcquireLock(context.Background(), "GR-UNKNOWN", "operator-1")
	assert.ErrorIs(t, err, service.ErrNotFound)
	grRepo.AssertExpectations(t)
}

func TestGoodsReceiptService_AcquireLock_AlreadyLockedByOther(t *testing.T) {
	svc, grRepo, _, _ := setupGRSvc()
	lockedBy := "other-operator"
	expiresAt := time.Now().Add(30 * time.Minute)
	gr := &models.GoodsReceipt{
		ID:            "GR-2025-001",
		LockedBy:      &lockedBy,
		LockExpiresAt: &expiresAt,
	}

	grRepo.On("GetGRByID", anyCtx, "GR-2025-001").Return(gr, nil)

	err := svc.AcquireLock(context.Background(), "GR-2025-001", "operator-1")
	assert.ErrorIs(t, err, service.ErrAlreadyLocked)
	grRepo.AssertExpectations(t)
}

func TestGoodsReceiptService_ReleaseLock_Success(t *testing.T) {
	svc, grRepo, _, _ := setupGRSvc()
	gr := &models.GoodsReceipt{ID: "GR-2025-001"}

	grRepo.On("GetGRByID", anyCtx, "GR-2025-001").Return(gr, nil)
	grRepo.On("UpdateGRFields", anyCtx, "GR-2025-001", map[string]interface{}{
		"locked_by":       nil,
		"lock_expires_at": nil,
	}).Return(nil)

	err := svc.ReleaseLock(context.Background(), "GR-2025-001")
	assert.NoError(t, err)
	grRepo.AssertExpectations(t)
}

func TestGoodsReceiptService_ReleaseLock_NotFound(t *testing.T) {
	svc, grRepo, _, _ := setupGRSvc()

	grRepo.On("GetGRByID", anyCtx, "GR-UNKNOWN").Return(nil, assert.AnError)

	err := svc.ReleaseLock(context.Background(), "GR-UNKNOWN")
	assert.ErrorIs(t, err, service.ErrNotFound)
	grRepo.AssertExpectations(t)
}
