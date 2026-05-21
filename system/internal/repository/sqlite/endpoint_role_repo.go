package sqlite

import (
	"context"

	"zeus-system-service/internal/models"

	"gorm.io/gorm"
)

type endpointRoleRepository struct {
	db *gorm.DB
}

func NewEndpointRoleRepository(db *gorm.DB) *endpointRoleRepository {
	return &endpointRoleRepository{db: db}
}

func (r *endpointRoleRepository) GetRequiredLevel(ctx context.Context, method, path string) (string, error) {
	var ep models.EndpointRole
	if err := r.db.WithContext(ctx).Where("method = ? AND path = ?", method, path).First(&ep).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", nil
		}
		return "", err
	}
	return ep.RequiredLevel, nil
}

func (r *endpointRoleRepository) GetAll(ctx context.Context) ([]models.EndpointRole, error) {
	var endpoints []models.EndpointRole
	if err := r.db.WithContext(ctx).Find(&endpoints).Error; err != nil {
		return nil, err
	}
	return endpoints, nil
}
