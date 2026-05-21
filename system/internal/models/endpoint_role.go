package models

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type EndpointRole struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	Method        string    `gorm:"not null;size:10" json:"method"`
	Path          string    `gorm:"not null;size:255" json:"path"`
	RequiredLevel string    `gorm:"not null;size:20" json:"required_level"`
}

func (e *EndpointRole) BeforeCreate(tx *gorm.DB) error {
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	return nil
}
