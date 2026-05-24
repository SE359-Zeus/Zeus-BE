package sqlite

import (
	"context"
	"strings"
	"time"

	"zeus-sales-service/internal/models"

	"github.com/google/uuid"
)

type salesOrderStatusRecord struct {
	ID         string    `gorm:"primaryKey;column:id"`
	Code       string    `gorm:"column:code;uniqueIndex"`
	Label      string    `gorm:"column:label"`
	SortOrder  int       `gorm:"column:sort_order"`
	IsTerminal bool      `gorm:"column:is_terminal"`
	CreatedAt  time.Time `gorm:"column:created_at"`
	UpdatedAt  time.Time `gorm:"column:updated_at"`
}

func (salesOrderStatusRecord) TableName() string { return "sales_order_status_lut" }

func (record salesOrderStatusRecord) toModel() models.SalesOrderStatusLUT {
	parsedID, _ := uuid.Parse(record.ID)
	return models.SalesOrderStatusLUT{
		ID:         parsedID,
		Code:       record.Code,
		Label:      record.Label,
		SortOrder:  record.SortOrder,
		IsTerminal: record.IsTerminal,
		CreatedAt:  record.CreatedAt,
		UpdatedAt:  record.UpdatedAt,
	}
}

func (repo *Repository) ListOrderStatuses(ctx context.Context) ([]models.SalesOrderStatusLUT, error) {
	var records []salesOrderStatusRecord
	if err := repo.db.WithContext(ctx).Order("sort_order ASC").Find(&records).Error; err != nil {
		return nil, err
	}
	statuses := make([]models.SalesOrderStatusLUT, 0, len(records))
	for _, record := range records {
		statuses = append(statuses, record.toModel())
	}
	return statuses, nil
}

func (repo *Repository) GetOrderStatusByID(ctx context.Context, id uuid.UUID) (*models.SalesOrderStatusLUT, error) {
	var record salesOrderStatusRecord
	if err := repo.db.WithContext(ctx).First(&record, "id = ?", id.String()).Error; err != nil {
		return nil, mapRecordError(err)
	}
	model := record.toModel()
	return &model, nil
}

func (repo *Repository) GetOrderStatusByCode(ctx context.Context, code string) (*models.SalesOrderStatusLUT, error) {
	var record salesOrderStatusRecord
	if err := repo.db.WithContext(ctx).Where("code = ?", strings.ToUpper(strings.TrimSpace(code))).First(&record).Error; err != nil {
		return nil, mapRecordError(err)
	}
	model := record.toModel()
	return &model, nil
}

func defaultStatuses() []models.SalesOrderStatusLUT {
	now := time.Now().UTC()
	codes := []struct {
		Code     string
		Label    string
		Order    int
		Terminal bool
	}{
		{models.SalesOrderStatusPendingCode, "Pending", 1, false},
		{models.SalesOrderStatusProcessingCode, "Processing", 2, false},
		{models.SalesOrderStatusDeliveringCode, "Delivering", 3, false},
		{models.SalesOrderStatusCompletedCode, "Completed", 4, true},
		{models.SalesOrderStatusCancelledCode, "Cancelled", 5, true},
	}
	statuses := make([]models.SalesOrderStatusLUT, 0, len(codes))
	for _, c := range codes {
		statuses = append(statuses, models.SalesOrderStatusLUT{
			ID:         salesStatusID(c.Code),
			Code:       c.Code,
			Label:      c.Label,
			SortOrder:  c.Order,
			IsTerminal: c.Terminal,
			CreatedAt:  now,
			UpdatedAt:  now,
		})
	}
	return statuses
}
