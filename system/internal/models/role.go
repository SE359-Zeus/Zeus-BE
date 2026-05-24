package models

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Role struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	Name        string    `gorm:"uniqueIndex;not null;size:50" json:"name"`
	DisplayName string    `gorm:"not null;size:100" json:"display_name"`
	Level       string    `gorm:"not null;size:20" json:"level"`
	Module      string    `gorm:"not null;size:20" json:"module"`
	Description string    `gorm:"size:255" json:"description"`
}

func (r *Role) BeforeCreate(tx *gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}
