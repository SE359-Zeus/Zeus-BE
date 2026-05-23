package sqlite

import (
	"context"
	"time"

	"zeus-sales-service/internal/models"
	rootrepo "zeus-sales-service/internal/repository"

	"github.com/google/uuid"
)

type salesOrderRecord struct {
	ID                 string                      `gorm:"primaryKey;column:id"`
	ClientID           string                      `gorm:"column:client_id;index"`
	ClientName         string                      `gorm:"column:client_name"`
	DestinationAddress string                      `gorm:"column:destination_address"`
	RequiredDate       time.Time                   `gorm:"column:required_date"`
	StatusID           string                      `gorm:"column:status_id;index"`
	Status             salesOrderStatusRecord      `gorm:"foreignKey:StatusID;references:ID"`
	TotalValue         float64                     `gorm:"column:total_value"`
	Locked             bool                        `gorm:"column:locked"`
	CreatedAt          time.Time                   `gorm:"column:created_at"`
	UpdatedAt          time.Time                   `gorm:"column:updated_at"`
	Items              []salesOrderItemRecord      `gorm:"foreignKey:OrderID;references:ID"`
	Reservation        *inventoryReservationRecord `gorm:"foreignKey:OrderID;references:ID"`
}

func (salesOrderRecord) TableName() string { return "sales_orders" }

func orderRecordFromModel(order *models.SalesOrder) *salesOrderRecord {
	record := &salesOrderRecord{
		ID:                 order.ID.String(),
		ClientID:           order.ClientID.String(),
		ClientName:         order.ClientName,
		DestinationAddress: order.DestinationAddress,
		RequiredDate:       order.RequiredDate,
		StatusID:           order.StatusID.String(),
		TotalValue:         order.TotalValue,
		Locked:             order.Locked,
		CreatedAt:          order.CreatedAt,
		UpdatedAt:          order.UpdatedAt,
	}
	if order.Status != nil {
		record.Status = salesOrderStatusRecord{
			ID:         order.Status.ID.String(),
			Code:       order.Status.Code,
			Label:      order.Status.Label,
			SortOrder:  order.Status.SortOrder,
			IsTerminal: order.Status.IsTerminal,
			CreatedAt:  order.Status.CreatedAt,
			UpdatedAt:  order.Status.UpdatedAt,
		}
	}
	return record
}

func (record salesOrderRecord) toModel() models.SalesOrder {
	parsedID, _ := uuid.Parse(record.ID)
	clientID, _ := uuid.Parse(record.ClientID)
	statusID, _ := uuid.Parse(record.StatusID)
	var status *models.SalesOrderStatusLUT
	if record.Status.ID != "" {
		statusModel := record.Status.toModel()
		status = &statusModel
	}
	return models.SalesOrder{
		ID:                 parsedID,
		ClientID:           clientID,
		ClientName:         record.ClientName,
		DestinationAddress: record.DestinationAddress,
		RequiredDate:       record.RequiredDate,
		StatusID:           statusID,
		Status:             status,
		TotalValue:         record.TotalValue,
		Locked:             record.Locked,
		CreatedAt:          record.CreatedAt,
		UpdatedAt:          record.UpdatedAt,
	}
}

func (repo *Repository) CreateOrder(ctx context.Context, order *models.SalesOrder) error {
	if order.ID == uuid.Nil {
		order.ID = uuid.New()
	}
	now := time.Now().UTC()
	if order.CreatedAt.IsZero() {
		order.CreatedAt = now
	}
	order.UpdatedAt = now
	return repo.db.WithContext(ctx).Create(orderRecordFromModel(order)).Error
}

func (repo *Repository) GetOrder(ctx context.Context, id uuid.UUID) (*models.SalesOrder, error) {
	var record salesOrderRecord
	if err := repo.db.WithContext(ctx).Preload("Status").First(&record, "id = ?", id.String()).Error; err != nil {
		return nil, mapRecordError(err)
	}
	model := record.toModel()
	return &model, nil
}

func (repo *Repository) ListOrders(ctx context.Context) ([]models.SalesOrder, error) {
	var records []salesOrderRecord
	if err := repo.db.WithContext(ctx).Preload("Status").Order("created_at ASC").Find(&records).Error; err != nil {
		return nil, err
	}
	orders := make([]models.SalesOrder, 0, len(records))
	for _, record := range records {
		orders = append(orders, record.toModel())
	}
	return orders, nil
}

func (repo *Repository) ListPendingOrders(ctx context.Context) ([]models.SalesOrder, error) {
	pendingStatus, err := repo.GetOrderStatusByCode(ctx, models.SalesOrderStatusPendingCode)
	if err != nil {
		return nil, err
	}
	var records []salesOrderRecord
	if err := repo.db.WithContext(ctx).Preload("Status").Where("status_id = ?", pendingStatus.ID.String()).Order("created_at ASC").Find(&records).Error; err != nil {
		return nil, err
	}
	orders := make([]models.SalesOrder, 0, len(records))
	for _, record := range records {
		orders = append(orders, record.toModel())
	}
	return orders, nil
}

func (repo *Repository) UpdateOrder(ctx context.Context, order *models.SalesOrder) error {
	order.UpdatedAt = time.Now().UTC()
	result := repo.db.WithContext(ctx).Model(&salesOrderRecord{}).Where("id = ?", order.ID.String()).Updates(map[string]any{
		"client_id":           order.ClientID.String(),
		"client_name":         order.ClientName,
		"destination_address": order.DestinationAddress,
		"required_date":       order.RequiredDate,
		"status_id":           order.StatusID.String(),
		"total_value":         order.TotalValue,
		"locked":              order.Locked,
		"updated_at":          order.UpdatedAt,
	})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return rootrepo.ErrNotFound
	}
	return nil
}
