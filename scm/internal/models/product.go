package models

import (
	"time"

	"github.com/google/uuid"
)

type Product struct {
	ID               uuid.UUID     `gorm:"type:uuid;primaryKey" json:"id"`
	ProductModelCode string        `gorm:"type:varchar;not null" json:"product_model_code"`
	CustomerID       *uuid.UUID    `gorm:"type:uuid" json:"customer_id,omitempty"`
	ProductName      string        `gorm:"type:varchar;not null" json:"product_name"`
	SerialNumber     string        `gorm:"type:varchar;not null" json:"serial_number"`
	CreatedAt        time.Time     `gorm:"not null" json:"created_at"`
	UpdatedAt        time.Time     `gorm:"not null" json:"updated_at"`
	DeletedAt        *time.Time    `json:"deleted_at,omitempty"`
	ProductModel     *ProductModel `gorm:"foreignKey:ProductModelCode;references:ModelCode" json:"product_model,omitempty"`
}
