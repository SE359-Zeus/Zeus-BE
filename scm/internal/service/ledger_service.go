package service

import (
	"context"

	"zeus-scm-service/internal/models"
	"zeus-scm-service/internal/pagination"
	"zeus-scm-service/internal/repository"

	"github.com/google/uuid"
)

type ILedgerService interface {
	RecordEntry(ctx context.Context, sku string, txnType models.LedgerTxnType, qtyChange int, operatorID, reference string, refType models.LedgerRefType, refID string) (*models.InventoryLedger, error)
	ListEntries(ctx context.Context, params pagination.Params, txnType, sku string) ([]models.InventoryLedger, *pagination.Meta, error)
	GetEntryByID(ctx context.Context, id string) (*models.InventoryLedger, error)
}

type ledgerService struct {
	repo repository.ILedgerRepository
}

func NewLedgerService(repo repository.ILedgerRepository) ILedgerService {
	return &ledgerService{repo: repo}
}

func (s *ledgerService) RecordEntry(ctx context.Context, sku string, txnType models.LedgerTxnType, qtyChange int, operatorID, reference string, refType models.LedgerRefType, refID string) (*models.InventoryLedger, error) {
	prevBalance, err := s.repo.GetLatestBalance(ctx, sku)
	if err != nil {
		return nil, err
	}
	operatorName := operatorNameFromContext(ctx)

	entry := &models.InventoryLedger{
		ID:             uuid.NewString(),
		SKU:            sku,
		Type:           txnType,
		QtyChange:      qtyChange,
		RunningBalance: prevBalance + qtyChange,
		Location:       "WH-A",
		OperatorID:     operatorID,
		OperatorName:   operatorName,
		Reference:      reference,
		ReferenceType:  refType,
		ReferenceID:    refID,
	}

	if err := s.repo.CreateEntry(ctx, entry); err != nil {
		return nil, err
	}
	return entry, nil
}

func (s *ledgerService) ListEntries(ctx context.Context, params pagination.Params, txnType, sku string) ([]models.InventoryLedger, *pagination.Meta, error) {
	return s.repo.ListEntries(ctx, params, txnType, sku)
}

func (s *ledgerService) GetEntryByID(ctx context.Context, id string) (*models.InventoryLedger, error) {
	return s.repo.GetEntryByID(ctx, id)
}
