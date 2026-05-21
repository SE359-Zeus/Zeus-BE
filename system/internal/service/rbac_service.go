package service

import (
	"context"
	"fmt"
	"log"

	"zeus-system-service/internal/repository"
)

var levelRank = map[string]int{
	"Administrator": 3,
	"Operator":      2,
	"Worker":        1,
}

type endpointRBACService struct {
	roleRepo        repository.RoleRepository
	endpointRepo    repository.EndpointRoleRepository
	cacheRepo       repository.EndpointRBACCacheRepository
}

func NewEndpointRBACService(
	roleRepo repository.RoleRepository,
	endpointRepo repository.EndpointRoleRepository,
	cacheRepo repository.EndpointRBACCacheRepository,
) *endpointRBACService {
	return &endpointRBACService{
		roleRepo:     roleRepo,
		endpointRepo: endpointRepo,
		cacheRepo:    cacheRepo,
	}
}

func (s *endpointRBACService) ValidateRole(ctx context.Context, role string) error {
	exists, err := s.roleRepo.Exists(ctx, role)
	if err != nil {
		return fmt.Errorf("failed to validate role: %w", err)
	}
	if !exists {
		return ErrInvalidRole
	}
	return nil
}

func (s *endpointRBACService) GetRequiredLevel(ctx context.Context, method, path string) (string, error) {
	level, err := s.cacheRepo.GetRequiredLevel(ctx, method, path)
	if err != nil {
		return "", err
	}
	if level != "" {
		return level, nil
	}

	level, err = s.endpointRepo.GetRequiredLevel(ctx, method, path)
	if err != nil {
		return "", fmt.Errorf("failed to get required level: %w", err)
	}
	return level, nil
}

func (s *endpointRBACService) GetRoleLevel(ctx context.Context, roleName string) (string, error) {
	level, err := s.cacheRepo.GetRoleLevel(ctx, roleName)
	if err != nil {
		return "", err
	}
	if level != "" {
		return level, nil
	}

	role, err := s.roleRepo.GetByName(ctx, roleName)
	if err != nil {
		return "", fmt.Errorf("failed to get role level: %w", err)
	}
	if role == nil {
		return "", ErrInvalidRole
	}
	return role.Level, nil
}

func (s *endpointRBACService) CanAccess(ctx context.Context, roleName, method, path string) (bool, error) {
	requiredLevel, err := s.GetRequiredLevel(ctx, method, path)
	if err != nil {
		return false, err
	}
	if requiredLevel == "" {
		return true, nil
	}

	roleLevel, err := s.GetRoleLevel(ctx, roleName)
	if err != nil {
		return false, err
	}

	return levelRank[roleLevel] >= levelRank[requiredLevel], nil
}

func (s *endpointRBACService) WarmCache(ctx context.Context) error {
	roles, err := s.roleRepo.GetAll(ctx)
	if err != nil {
		return fmt.Errorf("failed to load roles: %w", err)
	}

	roleLevels := make(map[string]string, len(roles))
	for _, r := range roles {
		roleLevels[r.Name] = r.Level
	}

	endpoints, err := s.endpointRepo.GetAll(ctx)
	if err != nil {
		return fmt.Errorf("failed to load endpoint roles: %w", err)
	}

	endpointLevels := make(map[string]string, len(endpoints))
	for _, e := range endpoints {
		key := fmt.Sprintf("rbac:endpoint:%s:%s", e.Method, e.Path)
		endpointLevels[key] = e.RequiredLevel
	}

	if err := s.cacheRepo.Warm(ctx, endpointLevels, roleLevels); err != nil {
		return fmt.Errorf("failed to warm RBAC cache: %w", err)
	}

	log.Printf("RBAC cache warmed: %d roles, %d endpoints", len(roles), len(endpoints))
	return nil
}
