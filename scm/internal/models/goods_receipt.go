package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type GRStatus string

const (
	GRStatusPending     GRStatus = "Pending"
	GRStatusInspected   GRStatus = "Inspected"
	GRStatusComplete    GRStatus = "Complete"
	GRStatusDiscrepancy GRStatus = "Discrepancy"
)

type GoodsReceipt struct {
	ID            string            `gorm:"type:varchar(50);primary_key"`
	PORef         string            `gorm:"type:varchar(50);not null"`
	VendorID      uuid.UUID         `gorm:"type:uuid;not null" json:"-"`
	VendorName    string            `gorm:"-" json:"vendor_name,omitempty"`
	Status        GRStatus          `gorm:"type:varchar(50);not null"`
	State         GoodsReceiptState `gorm:"foreignKey:Status;references:Name"`
	ArrivalDate   time.Time         `gorm:"not null"`
	OperatorID    string            `gorm:"type:varchar(100)" json:"-"`
	OperatorName  string            `gorm:"type:varchar(255)" json:"operator_name,omitempty"`
	LockedBy      *string           `gorm:"type:varchar(100)"` // Optional lock owner
	LockExpiresAt *time.Time        `gorm:""`                  // Lock expiration timestamp
	CreatedAt     time.Time         `gorm:"autoCreateTime"`
	UpdatedAt     time.Time         `gorm:"autoUpdateTime"`
	DeletedAt     gorm.DeletedAt    `gorm:"index"`

	LineItems []GRLineItem `gorm:"foreignKey:GRID"`
	Vendor    *Supplier    `gorm:"foreignKey:VendorID;references:ID" json:"-"`
}

type GRLineItem struct {
	ID             uuid.UUID  `gorm:"type:uuid;primary_key"`
	GRID           string     `gorm:"type:varchar(50);not null"`
	SKU            string     `gorm:"type:varchar(100);not null"`
	Name           string     `gorm:"type:varchar(255);not null"`
	OrderedQty     int        `gorm:"not null"`
	ReceivedQty    *int       `gorm:""` // Pointer because it must be manually typed (blind receiving)
	DefectiveQty   *int       `gorm:""`
	AgingSensitive bool       `gorm:"not null;default:false"`
	ProductionDate *time.Time `gorm:""` // Required if AgingSensitive is true
	AgingLabel     string     `gorm:"type:varchar(255)"`
}
