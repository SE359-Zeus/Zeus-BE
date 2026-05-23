package service_test

import (
	"context"
	"errors"
	"testing"

	"zeus-system-service/internal/models"
	"zeus-system-service/internal/repository"
	"zeus-system-service/internal/service"

	"github.com/stretchr/testify/assert"
)

type staticRoleRepo struct {
	role *models.Role
}

func (r staticRoleRepo) Create(ctx context.Context, role *models.Role) error { return nil }
func (r staticRoleRepo) GetAll(ctx context.Context) ([]models.Role, error) {
	return []models.Role{*r.role}, nil
}
func (r staticRoleRepo) GetByName(ctx context.Context, name string) (*models.Role, error) {
	return r.role, nil
}
func (r staticRoleRepo) Exists(ctx context.Context, name string) (bool, error) { return true, nil }

type staticActionTypeRepo struct {
	exists bool
}

func (r staticActionTypeRepo) GetAll(ctx context.Context) ([]models.ActionTypeEntry, error) {
	return []models.ActionTypeEntry{{Name: "CREATE"}}, nil
}
func (r staticActionTypeRepo) Exists(ctx context.Context, name string) (bool, error) {
	return r.exists, nil
}

type failingActionTypeCacheRepo struct{}

func (r failingActionTypeCacheRepo) Warm(ctx context.Context, names []string) error {
	return errors.New("cache unavailable")
}
func (r failingActionTypeCacheRepo) IsValid(ctx context.Context, name string) (bool, error) {
	return false, errors.New("cache unavailable")
}

func TestEndpointRBACService_ValidateRole(t *testing.T) {
	rbacSvc := service.NewEndpointRBACService(
		staticRoleRepo{role: &models.Role{Name: "Administrator", Level: "Administrator"}},
	)

	err := rbacSvc.ValidateRole(context.Background(), "Administrator")
	assert.NoError(t, err)
}

func TestActionTypeService_IsValid_FallsBackWhenCacheUnavailable(t *testing.T) {
	actionTypeSvc := service.NewActionTypeService(
		staticActionTypeRepo{exists: true},
		failingActionTypeCacheRepo{},
	)

	ok, err := actionTypeSvc.IsValid(context.Background(), models.ActionType("CREATE"))
	assert.NoError(t, err)
	assert.True(t, ok)
}

var _ repository.RoleRepository = staticRoleRepo{}
var _ repository.ActionTypeRepository = staticActionTypeRepo{}
var _ repository.ActionTypeCacheRepository = failingActionTypeCacheRepo{}
