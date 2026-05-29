package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type POStatus string

const (
	POStatusDraft     POStatus = "Draft"
	POStatusApproved  POStatus = "Approved"
	POStatusInTransit POStatus = "In Transit"
	POStatusReceived  POStatus = "Received"
	POStatusPartial   POStatus = "Partial"
	POStatusVoid      POStatus = "Void"
)

type PurchaseOrder struct {
	ID               string             `gorm:"type:varchar(50);primary_key"`
	VendorID         uuid.UUID          `gorm:"type:uuid;not null"`
	TargetBuild      string             `gorm:"type:varchar(255)"`
	Status           POStatus           `gorm:"type:varchar(50);not null"`
	State            PurchaseOrderState `gorm:"foreignKey:Status;references:Name"`
	TotalValue       float64            `gorm:"not null"`
	PaymentTerms     string             `gorm:"type:varchar(100)"`
	ExpectedDelivery time.Time          `gorm:"not null"`
	Notes            string             `gorm:"type:text"`
	CreatedAt        time.Time          `gorm:"autoCreateTime"`
	UpdatedAt        time.Time          `gorm:"autoUpdateTime"`
	DeletedAt        gorm.DeletedAt     `gorm:"index"`

	LineItems []POLineItem `gorm:"foreignKey:POID"`
}

type POLineItem struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key"`
	POID        string    `gorm:"type:varchar(50);not null"`
	SKU         string    `gorm:"type:varchar(100);not null"`
	Description string    `gorm:"type:varchar(255);not null"`
	OrderedQty  int       `gorm:"not null"`
	ReceivedQty int       `gorm:"not null;default:0"`
	UnitPrice   float64   `gorm:"not null"`
}
