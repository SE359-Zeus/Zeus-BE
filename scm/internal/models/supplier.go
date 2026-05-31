package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SupplierTier string

const (
	SupplierTier1 SupplierTier = "Tier 1"
	SupplierTier2 SupplierTier = "Tier 2"
	SupplierTier3 SupplierTier = "Tier 3"
)

type Supplier struct {
	ID           uuid.UUID      `gorm:"type:uuid;primary_key"`
	Name         string         `gorm:"type:varchar(255);not null"`
	Contact      string         `gorm:"type:varchar(255);not null"`
	Tier         SupplierTier   `gorm:"type:varchar(50);not null"`
	LeadTimeDays int            `gorm:"not null"`
	QualityScore float64        `gorm:"not null"`
	OnTimeRate   float64        `gorm:"not null"`
	CreatedAt    time.Time      `gorm:"autoCreateTime"`
	UpdatedAt    time.Time      `gorm:"autoUpdateTime"`
	DeletedAt    gorm.DeletedAt `gorm:"index"`

	SkuMappings []SkuMapping `gorm:"foreignKey:SupplierID"`
}

type SkuMapping struct {
	ID           uuid.UUID `gorm:"type:uuid;primary_key"`
	SupplierID   uuid.UUID `gorm:"type:uuid;not null"`
	Supplier     Supplier  `gorm:"foreignKey:SupplierID;references:ID"`
	SKU          string    `gorm:"type:varchar(100);not null"` // Foreign reference to ComponentStock.SKU
	Name         string    `gorm:"type:varchar(255);not null"`
	UnitPrice    float64   `gorm:"not null"`
	LeadTimeDays int       `gorm:"not null"`
	MinOrderQty  int       `gorm:"not null;default:1"`
	CreatedAt    time.Time `gorm:"autoCreateTime"`
	UpdatedAt    time.Time `gorm:"autoUpdateTime"`
}

type ShortageSummaryDTO struct {
	STT          int    `json:"stt"`
	SKU          string `json:"sku"`
	ReqQty       int    `json:"req_qty"`
	BestSupplier string `json:"best_supplier"`
}
