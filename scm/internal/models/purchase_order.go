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
	ID           string             `gorm:"type:varchar(50);primary_key" json:"id"`
	VendorID     uuid.UUID          `gorm:"type:uuid;not null;column:vendor_id" json:"supplier_id"`
	VendorID_Old uuid.UUID          `gorm:"-" json:"vendor_id,omitempty"`
	VendorName   string             `gorm:"-" json:"supplier_name,omitempty"`
	VendorName_O string             `gorm:"-" json:"vendor_name,omitempty"`
	Status       POStatus           `gorm:"type:varchar(50);not null" json:"status"`
	State        PurchaseOrderState `gorm:"foreignKey:Status;references:Name" json:"-"`
	TotalValue   float64            `gorm:"not null" json:"total_value"`
	PaymentTerms string             `gorm:"type:varchar(100)" json:"payment_terms,omitempty"`
	Notes        string             `gorm:"type:text" json:"notes,omitempty"`
	CreatedAt    time.Time          `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time          `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt    gorm.DeletedAt     `gorm:"index" json:"-"`

	LineItems []POLineItem `gorm:"foreignKey:POID" json:"line_items,omitempty"`
	Vendor    *Supplier    `gorm:"foreignKey:VendorID;references:ID" json:"-"`
}

type POLineItem struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key" json:"id"`
	POID        string    `gorm:"type:varchar(50);not null" json:"po_id"`
	SKU         string    `gorm:"type:varchar(100);not null" json:"sku"`
	Description string    `gorm:"type:varchar(255);not null" json:"description"`
	OrderedQty  int       `gorm:"not null" json:"ordered_qty"`
	ReceivedQty int       `gorm:"not null;default:0" json:"received_qty"`
	UnitPrice   float64   `gorm:"not null" json:"unit_price"`
}
