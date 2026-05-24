package service

import (
	"context"
	"fmt"
	"log"

	"zeus-system-service/internal/models"
	"zeus-system-service/internal/repository"
)

type actionTypeService struct {
	repo      repository.ActionTypeRepository
	cacheRepo repository.ActionTypeCacheRepository
}

func NewActionTypeService(
	repo repository.ActionTypeRepository,
	cacheRepo repository.ActionTypeCacheRepository,
) *actionTypeService {
	return &actionTypeService{
		repo:      repo,
		cacheRepo: cacheRepo,
	}
}

func (s *actionTypeService) IsValid(ctx context.Context, name models.ActionType) (bool, error) {
	ok, err := s.cacheRepo.IsValid(ctx, string(name))
	if err == nil && ok {
		return true, nil
	}

	return s.repo.Exists(ctx, string(name))
}

func (s *actionTypeService) WarmCache(ctx context.Context) error {
	entries, err := s.repo.GetAll(ctx)
	if err != nil {
		return fmt.Errorf("failed to load action types: %w", err)
	}

	names := make([]string, len(entries))
	for i, e := range entries {
		names[i] = e.Name
	}

	if err := s.cacheRepo.Warm(ctx, names); err != nil {
		return fmt.Errorf("failed to warm action type cache: %w", err)
	}

	log.Printf("Action type cache warmed: %d types", len(names))
	return nil
}
