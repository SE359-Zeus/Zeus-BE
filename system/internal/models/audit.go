package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ActionType string

type AuditLog struct {
	ID              uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	Timestamp       time.Time      `gorm:"not null;index" json:"timestamp"`
	UserID          uuid.UUID      `gorm:"type:uuid;not null;index" json:"user_id"`
	UserEmail       string         `gorm:"not null;size:255" json:"user_email"`
	ActionType      ActionType     `gorm:"not null;size:20;index" json:"action_type"`
	TargetResource  string         `gorm:"not null;size:255" json:"target_resource"`
	Details         string         `gorm:"type:text" json:"details,omitempty"`
	IPAddress       string         `gorm:"size:45" json:"ip_address,omitempty"`
	IsSecurityEvent bool           `gorm:"not null;default:false;index" json:"is_security_event"`
	CreatedAt       time.Time      `json:"-"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`
}

func (a *AuditLog) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	if a.Timestamp.IsZero() {
		a.Timestamp = time.Now()
	}
	return nil
}

type IngestAuditRequest struct {
	UserID          uuid.UUID  `json:"user_id" binding:"required"`
	UserEmail       string     `json:"user_email" binding:"required"`
	ActionType      ActionType `json:"action_type" binding:"required"`
	TargetResource  string     `json:"target_resource" binding:"required"`
	Details         string     `json:"details,omitempty"`
	IPAddress       string     `json:"ip_address,omitempty"`
	IsSecurityEvent bool       `json:"is_security_event,omitempty"`
}

type AuditFilter struct {
	ActionType *ActionType `json:"action_type,omitempty"`
	UserID     *uuid.UUID  `json:"user_id,omitempty"`
	StartDate  *time.Time  `json:"start_date,omitempty"`
	EndDate    *time.Time  `json:"end_date,omitempty"`
	Limit      int         `json:"limit,omitempty"`
	Offset     int         `json:"offset,omitempty"`
}

type AuditMetrics struct {
	LoginsToday          int64 `json:"logins_today"`
	SecurityEvents       int64 `json:"security_events"`
	ModificationVelocity int64 `json:"modification_velocity"`
}
