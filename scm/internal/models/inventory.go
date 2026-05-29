package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ComponentStatus string

const (
	ComponentStatusInStock      ComponentStatus = "In Stock"
	ComponentStatusLowStock     ComponentStatus = "Low Stock"
	ComponentStatusOutOfStock   ComponentStatus = "Out of Stock"
	ComponentStatusDiscontinued ComponentStatus = "Discontinued"
)

// ComponentStock represents the inventory ledger entity derived from the SCM UI requirements
// This complements the reference Product/Part entities by providing warehouse logistics data.
type ComponentStock struct {
	SKU               string              `gorm:"type:varchar(100);primary_key" json:"sku"`
	Name              string              `gorm:"type:varchar(255);not null" json:"name"`
	Category          string              `gorm:"type:varchar(100);not null" json:"category"`
	StockQty          int                 `gorm:"not null;default:0" json:"stock_qty"`
	ReorderPoint      int                 `gorm:"not null;default:0" json:"reorder_point"`
	UnitCost          float64             `gorm:"not null;default:0.0" json:"unit_cost"`
	Status            ComponentStatus     `gorm:"type:varchar(50);not null" json:"status"`
	State             ComponentStockState `gorm:"foreignKey:Status;references:Name" json:"-"`
	PrimarySupplierID uuid.UUID           `gorm:"type:uuid" json:"primary_supplier_id"`
	PrimarySupplier   string              `gorm:"-" json:"primary_supplier,omitempty"`
	LeadTimeDays      int                 `gorm:"not null;default:0" json:"lead_time_days"`
	Location          string              `gorm:"type:varchar(255)" json:"location"`
	CreatedAt         time.Time           `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt         time.Time           `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt         gorm.DeletedAt      `gorm:"index" json:"-"`
}
