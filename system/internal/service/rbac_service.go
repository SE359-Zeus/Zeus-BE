package service

import (
	"context"
	"fmt"
	"log"

	"zeus-system-service/internal/repository"
)

type endpointRBACService struct {
	roleRepo repository.RoleRepository
}

func NewEndpointRBACService(roleRepo repository.RoleRepository) *endpointRBACService {
	return &endpointRBACService{
		roleRepo: roleRepo,
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

func (s *endpointRBACService) WarmCache(ctx context.Context) error {
	roles, err := s.roleRepo.GetAll(ctx)
	if err != nil {
		return fmt.Errorf("failed to load roles: %w", err)
	}

	log.Printf("RBAC role validation ready: %d roles loaded", len(roles))
	return nil
}
