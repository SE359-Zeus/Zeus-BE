package service

import (
	"context"
	"fmt"
	"log"
	"sync"

	"zeus-system-service/internal/models"
	"zeus-system-service/internal/repository"
)

type actionTypeService struct {
	repo       repository.ActionTypeRepository
	cacheRepo  repository.ActionTypeCacheRepository
	localCache map[models.ActionType]bool
	mu         sync.RWMutex
}

func NewActionTypeService(
	repo repository.ActionTypeRepository,
	cacheRepo repository.ActionTypeCacheRepository,
) *actionTypeService {
	return &actionTypeService{
		repo:       repo,
		cacheRepo:  cacheRepo,
		localCache: make(map[models.ActionType]bool),
	}
}

func (s *actionTypeService) IsValid(ctx context.Context, name models.ActionType) (bool, error) {
	s.mu.RLock()
	val, found := s.localCache[name]
	s.mu.RUnlock()
	if found {
		return val, nil
	}

	ok, err := s.cacheRepo.IsValid(ctx, string(name))
	if err == nil && ok {
		s.mu.Lock()
		s.localCache[name] = true
		s.mu.Unlock()
		return true, nil
	}

	exists, err := s.repo.Exists(ctx, string(name))
	if err == nil && exists {
		s.mu.Lock()
		s.localCache[name] = true
		s.mu.Unlock()
		return true, nil
	}

	return false, err
}

func (s *actionTypeService) WarmCache(ctx context.Context) error {
	entries, err := s.repo.GetAll(ctx)
	if err != nil {
		return fmt.Errorf("failed to load action types: %w", err)
	}

	s.mu.Lock()
	s.localCache = make(map[models.ActionType]bool, len(entries))
	names := make([]string, len(entries))
	for i, e := range entries {
		names[i] = e.Name
		s.localCache[models.ActionType(e.Name)] = true
	}
	s.mu.Unlock()

	if err := s.cacheRepo.Warm(ctx, names); err != nil {
		return fmt.Errorf("failed to warm action type cache: %w", err)
	}

	log.Printf("Action type cache warmed: %d types", len(names))
	return nil
}
