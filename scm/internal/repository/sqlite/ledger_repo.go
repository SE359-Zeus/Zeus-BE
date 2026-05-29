package sqlite

import (
	"context"

	"zeus-scm-service/internal/models"
	"zeus-scm-service/internal/pagination"
	"zeus-scm-service/internal/repository"

	"gorm.io/gorm"
)

type ledgerRepository struct {
	db *gorm.DB
}

func NewLedgerRepository(db *gorm.DB) repository.ILedgerRepository {
	return &ledgerRepository{db: db}
}

func (r *ledgerRepository) CreateEntry(ctx context.Context, entry *models.InventoryLedger) error {
	return r.db.WithContext(ctx).Create(entry).Error
}

func (r *ledgerRepository) ListEntries(ctx context.Context, params pagination.Params, txnType, sku string) ([]models.InventoryLedger, *pagination.Meta, error) {
	query := r.db.WithContext(ctx).Model(&models.InventoryLedger{})
	if txnType != "" {
		query = query.Where("type = ?", txnType)
	}
	if sku != "" {
		query = query.Where("sku = ?", sku)
	}
	var entries []models.InventoryLedger
	meta, err := pagination.Paginate(query, params, &entries, "created_at", "sku", "type")
	if err != nil {
		return nil, nil, err
	}
	return entries, meta, nil
}

func (r *ledgerRepository) GetEntryByID(ctx context.Context, id string) (*models.InventoryLedger, error) {
	var entry models.InventoryLedger
	if err := r.db.WithContext(ctx).First(&entry, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &entry, nil
}

func (r *ledgerRepository) GetLatestBalance(ctx context.Context, sku string) (int, error) {
	var entry models.InventoryLedger
	if err := r.db.WithContext(ctx).Where("sku = ?", sku).Order("created_at DESC, id DESC").First(&entry).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return 0, nil
		}
		return 0, err
	}
	return entry.RunningBalance, nil
}
