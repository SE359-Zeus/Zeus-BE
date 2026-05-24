package sqlite

import (
	"context"
	"time"

	"zeus-system-service/internal/models"
	"zeus-system-service/internal/repository"

	"gorm.io/gorm"
)

type auditRepo struct {
	db *gorm.DB
}

func NewAuditRepository(db *gorm.DB) repository.AuditRepository {
	return &auditRepo{db: db}
}

func (r *auditRepo) Insert(ctx context.Context, log *models.AuditLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

func (r *auditRepo) Query(ctx context.Context, filter models.AuditFilter, page, limit int) ([]models.AuditLog, int64, error) {
	query := r.db.WithContext(ctx).Model(&models.AuditLog{})
	if filter.ActionType != nil {
		query = query.Where("action_type = ?", *filter.ActionType)
	}
	if filter.UserID != nil {
		query = query.Where("user_id = ?", *filter.UserID)
	}
	if filter.StartDate != nil {
		query = query.Where("timestamp >= ?", *filter.StartDate)
	}
	if filter.EndDate != nil {
		query = query.Where("timestamp <= ?", *filter.EndDate)
	}

	var totalRows int64
	if err := query.Count(&totalRows).Error; err != nil {
		return nil, 0, err
	}

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 15
	}

	var logs []models.AuditLog
	offset := (page - 1) * limit
	if err := query.Offset(offset).Limit(limit).Order("timestamp DESC").Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, totalRows, nil
}

func (r *auditRepo) CountByAction(ctx context.Context, actionType models.ActionType, start, end time.Time) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.AuditLog{}).
		Where("action_type = ?", actionType).
		Where("timestamp >= ?", start).
		Where("timestamp <= ?", end).
		Count(&count).Error
	return count, err
}
