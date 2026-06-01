package service_test

import (
	"context"
	"testing"

	"zeus-scm-service/internal/models"
	"zeus-scm-service/internal/repository"
	"zeus-scm-service/internal/service"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupPOSvc() (service.IPOService, *repository.MockPORepository, *repository.MockStockRepository, *repository.MockGoodsReceiptRepository) {
	poRepo := new(repository.MockPORepository)
	stockRepo := new(repository.MockStockRepository)
	grRepo := new(repository.MockGoodsReceiptRepository)
	svc := service.NewPOService(poRepo, stockRepo, grRepo, "")
	return svc, poRepo, stockRepo, grRepo
}

func TestPOService_CreateDraft_Success(t *testing.T) {
	svc, poRepo, _, _ := setupPOSvc()
	vendorID := uuid.New()

	poRepo.On("FindPOByVendorAndStatuses", anyCtx, vendorID, mock.Anything).Return(nil, assert.AnError)
	poRepo.On("CountPOsByYearPattern", anyCtx, mock.AnythingOfType("int"), "PO-%d-%%").Return(int64(0), nil)
	poRepo.On("CreatePO", anyCtx, mock.MatchedBy(func(po *models.PurchaseOrder) bool {
		return po.VendorID == vendorID && po.Status == models.POStatusDraft
	})).Return(nil)

	po, err := svc.CreateDraft(context.Background(), vendorID)
	assert.NoError(t, err)
	assert.NotNil(t, po)
	assert.Equal(t, models.POStatusDraft, po.Status)
	assert.Equal(t, vendorID, po.VendorID)
	poRepo.AssertExpectations(t)
}

func TestPOService_CreateDraft_MonoVendorViolation(t *testing.T) {
	svc, poRepo, _, _ := setupPOSvc()
	vendorID := uuid.New()
	existingPO := &models.PurchaseOrder{ID: "PO-2025-001", VendorID: vendorID, Status: models.POStatusDraft}

	poRepo.On("FindPOByVendorAndStatuses", anyCtx, vendorID, mock.Anything).Return(existingPO, nil)

	po, err := svc.CreateDraft(context.Background(), vendorID)
	assert.ErrorIs(t, err, service.ErrMonoVendorViolation)
	assert.Nil(t, po)
	poRepo.AssertExpectations(t)
}

func TestPOService_ApprovePO_Success(t *testing.T) {
	svc, poRepo, _, _ := setupPOSvc()
	poID := "PO-2025-001"

	po := &models.PurchaseOrder{
		ID:     poID,
		Status: models.POStatusDraft,
	}
	lineItems := []models.POLineItem{
		{SKU: "SOC-001", OrderedQty: 10, UnitPrice: 100.0},
		{SKU: "SOC-002", OrderedQty: 5, UnitPrice: 200.0},
	}

	poRepo.On("GetPOByID", anyCtx, poID).Return(po, nil)
	poRepo.On("GetPOLineItemsByPOID", anyCtx, poID).Return(lineItems, nil)
	poRepo.On("SavePO", anyCtx, mock.MatchedBy(func(p *models.PurchaseOrder) bool {
		return p.Status == models.POStatusApproved && p.TotalValue == 2000.0
	})).Return(nil)

	err := svc.ApprovePO(context.Background(), poID)
	assert.NoError(t, err)
	poRepo.AssertExpectations(t)
}

func TestPOService_ApprovePO_NotFound(t *testing.T) {
	svc, poRepo, _, _ := setupPOSvc()

	poRepo.On("GetPOByID", anyCtx, "PO-UNKNOWN").Return(nil, assert.AnError)

	err := svc.ApprovePO(context.Background(), "PO-UNKNOWN")
	assert.ErrorIs(t, err, service.ErrNotFound)
	poRepo.AssertExpectations(t)
}

func TestPOService_ApprovePO_InvalidTransition(t *testing.T) {
	svc, poRepo, _, _ := setupPOSvc()
	po := &models.PurchaseOrder{
		ID:     "PO-2025-001",
		Status: models.POStatusApproved,
	}

	poRepo.On("GetPOByID", anyCtx, "PO-2025-001").Return(po, nil)

	err := svc.ApprovePO(context.Background(), "PO-2025-001")
	assert.ErrorIs(t, err, service.ErrInvalidTransition)
	poRepo.AssertExpectations(t)
}

func TestPOService_TransitionState_Success(t *testing.T) {
	svc, poRepo, _, grRepo := setupPOSvc()
	po := &models.PurchaseOrder{
		ID:     "PO-2025-001",
		Status: models.POStatusApproved,
	}

	poRepo.On("GetPOByID", anyCtx, "PO-2025-001").Return(po, nil)
	grRepo.On("CountByPOIDAndNotStatus", anyCtx, "PO-2025-001", models.GRStatusComplete).Return(int64(0), nil)
	poRepo.On("UpdatePOStatus", anyCtx, "PO-2025-001", models.POStatusInTransit).Return(nil)

	err := svc.TransitionState(context.Background(), "PO-2025-001", models.POStatusInTransit)
	assert.NoError(t, err)
	poRepo.AssertExpectations(t)
	grRepo.AssertExpectations(t)
}

func TestPOService_TransitionState_Regression(t *testing.T) {
	svc, poRepo, _, _ := setupPOSvc()
	po := &models.PurchaseOrder{
		ID:     "PO-2025-001",
		Status: models.POStatusInTransit,
	}

	poRepo.On("GetPOByID", anyCtx, "PO-2025-001").Return(po, nil)

	err := svc.TransitionState(context.Background(), "PO-2025-001", models.POStatusDraft)
	assert.ErrorIs(t, err, service.ErrStateRegression)
	poRepo.AssertExpectations(t)
}

func TestPOService_TransitionState_NotFound(t *testing.T) {
	svc, poRepo, _, _ := setupPOSvc()

	poRepo.On("GetPOByID", anyCtx, "PO-UNKNOWN").Return(nil, assert.AnError)

	err := svc.TransitionState(context.Background(), "PO-UNKNOWN", models.POStatusApproved)
	assert.ErrorIs(t, err, service.ErrNotFound)
	poRepo.AssertExpectations(t)
}

func TestPOService_TransitionState_VoidFromDraft(t *testing.T) {
	svc, poRepo, _, _ := setupPOSvc()
	po := &models.PurchaseOrder{
		ID:     "PO-2025-001",
		Status: models.POStatusDraft,
	}

	poRepo.On("GetPOByID", anyCtx, "PO-2025-001").Return(po, nil)
	poRepo.On("UpdatePOStatus", anyCtx, "PO-2025-001", models.POStatusVoid).Return(nil)

	err := svc.TransitionState(context.Background(), "PO-2025-001", models.POStatusVoid)
	assert.NoError(t, err)
	poRepo.AssertExpectations(t)
}

func TestPOService_TransitionState_BlockedByIncompleteGRs(t *testing.T) {
	svc, poRepo, _, grRepo := setupPOSvc()
	po := &models.PurchaseOrder{
		ID:     "PO-2025-001",
		Status: models.POStatusInTransit,
	}

	poRepo.On("GetPOByID", anyCtx, "PO-2025-001").Return(po, nil)
	grRepo.On("CountByPOIDAndNotStatus", anyCtx, "PO-2025-001", models.GRStatusComplete).Return(int64(2), nil)

	err := svc.TransitionState(context.Background(), "PO-2025-001", models.POStatusReceived)
	assert.ErrorIs(t, err, service.ErrIncompleteGRs)
	poRepo.AssertExpectations(t)
	grRepo.AssertExpectations(t)
}

func TestPOService_TransitionState_AllowsVoidWithIncompleteGRs(t *testing.T) {
	svc, poRepo, _, _ := setupPOSvc()
	po := &models.PurchaseOrder{
		ID:     "PO-2025-001",
		Status: models.POStatusInTransit,
	}

	poRepo.On("GetPOByID", anyCtx, "PO-2025-001").Return(po, nil)
	poRepo.On("UpdatePOStatus", anyCtx, "PO-2025-001", models.POStatusVoid).Return(nil)

	err := svc.TransitionState(context.Background(), "PO-2025-001", models.POStatusVoid)
	assert.NoError(t, err)
	poRepo.AssertExpectations(t)
}

func TestPOService_TransitionState_PartialToReceived(t *testing.T) {
	svc, poRepo, _, grRepo := setupPOSvc()
	po := &models.PurchaseOrder{
		ID:     "PO-2025-001",
		Status: models.POStatusPartial,
	}

	poRepo.On("GetPOByID", anyCtx, "PO-2025-001").Return(po, nil)
	grRepo.On("CountByPOIDAndNotStatus", anyCtx, "PO-2025-001", models.GRStatusComplete).Return(int64(0), nil)
	poRepo.On("UpdatePOStatus", anyCtx, "PO-2025-001", models.POStatusReceived).Return(nil)

	err := svc.TransitionState(context.Background(), "PO-2025-001", models.POStatusReceived)
	assert.NoError(t, err)
	poRepo.AssertExpectations(t)
	grRepo.AssertExpectations(t)
}

func TestPOService_TransitionState_PartialToInTransit(t *testing.T) {
	svc, poRepo, _, grRepo := setupPOSvc()
	po := &models.PurchaseOrder{
		ID:     "PO-2025-001",
		Status: models.POStatusPartial,
	}

	poRepo.On("GetPOByID", anyCtx, "PO-2025-001").Return(po, nil)
	grRepo.On("CountByPOIDAndNotStatus", anyCtx, "PO-2025-001", models.GRStatusComplete).Return(int64(0), nil)
	poRepo.On("UpdatePOStatus", anyCtx, "PO-2025-001", models.POStatusInTransit).Return(nil)

	err := svc.TransitionState(context.Background(), "PO-2025-001", models.POStatusInTransit)
	assert.NoError(t, err)
	poRepo.AssertExpectations(t)
	grRepo.AssertExpectations(t)
}
