package sqlite

import (
	"context"
	"time"

	"zeus-system-service/internal/models"
	"zeus-system-service/internal/repository"

	"gorm.io/gorm"
)

type sessionRepo struct {
	db *gorm.DB
}

func NewSessionRepository(db *gorm.DB) repository.SessionRepository {
	return &sessionRepo{db: db}
}

func (r *sessionRepo) Create(ctx context.Context, session *models.Session) error {
	return r.db.WithContext(ctx).Create(session).Error
}

func (r *sessionRepo) GetByJTI(ctx context.Context, jti string) (*models.Session, error) {
	var session models.Session
	err := r.db.WithContext(ctx).
		Where("jti = ? AND expires_at > ?", jti, time.Now()).
		First(&session).Error
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func (r *sessionRepo) DeleteByJTI(ctx context.Context, jti string) error {
	result := r.db.WithContext(ctx).Where("jti = ?", jti).Delete(&models.Session{})
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (r *sessionRepo) DeleteByUserID(ctx context.Context, userID string) error {
	result := r.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&models.Session{})
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (r *sessionRepo) DeleteExpired(ctx context.Context) error {
	return r.db.WithContext(ctx).
		Where("expires_at <= ?", time.Now()).
		Delete(&models.Session{}).Error
}

func (r *sessionRepo) ListActive(ctx context.Context) ([]models.Session, error) {
	var sessions []models.Session
	err := r.db.WithContext(ctx).
		Where("expires_at > ?", time.Now()).
		Find(&sessions).Error
	if err != nil {
		return nil, err
	}
	return sessions, nil
}
