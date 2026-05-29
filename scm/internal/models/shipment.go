package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ShipmentStatus string

const (
	ShipmentStatusScheduled ShipmentStatus = "Scheduled"
	ShipmentStatusInTransit ShipmentStatus = "In Transit"
	ShipmentStatusDelivered ShipmentStatus = "Delivered"
	ShipmentStatusDelayed   ShipmentStatus = "Delayed"
)

type Shipment struct {
	ID         string         `gorm:"type:varchar(50);primary_key"`
	PORef      string         `gorm:"type:varchar(50);not null"`
	SupplierID uuid.UUID      `gorm:"type:uuid;not null"`
	Status     ShipmentStatus `gorm:"type:varchar(50);not null"`
	State      ShipmentState  `gorm:"foreignKey:Status;references:Name"`
	Carrier    string         `gorm:"type:varchar(100)"`
	TrackingNo string         `gorm:"type:varchar(100)"`
	Origin     string         `gorm:"type:varchar(255)"`
	ShipDate   time.Time      `gorm:""`
	ETA        time.Time      `gorm:""`
	CreatedAt  time.Time      `gorm:"autoCreateTime"`
	UpdatedAt  time.Time      `gorm:"autoUpdateTime"`
	DeletedAt  gorm.DeletedAt `gorm:"index"`

	Items    []ShipmentItem `gorm:"foreignKey:ShipmentID"`
	Supplier *Supplier      `gorm:"foreignKey:SupplierID;references:ID"`
}

type ShipmentItem struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key"`
	ShipmentID  string    `gorm:"type:varchar(50);not null"`
	SKU         string    `gorm:"type:varchar(100);not null"`
	Description string    `gorm:"type:varchar(255);not null"`
	Qty         int       `gorm:"not null"`
}
