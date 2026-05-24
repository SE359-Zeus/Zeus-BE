package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"zeus-sales-service/internal/middlewares"
	"zeus-sales-service/internal/models"
	"zeus-sales-service/internal/repository"

	"github.com/google/uuid"
)

type ClientService struct {
	repo  repository.DbRepository
	cache repository.CacheRepository
}

func NewClientService(repo repository.DbRepository, cache repository.CacheRepository) *ClientService {
	return &ClientService{repo: repo, cache: cache}
}

func (svc *ClientService) ResolveOrCreateClient(ctx context.Context, name string, destination string, tier models.ClientTier) (*models.Client, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("%w: client name is required", middlewares.ErrValidation)
	}
	if tier == "" {
		tier = models.ClientTierB2C
	}
	client, err := svc.repo.GetClientByName(ctx, name)
	if err == nil {
		return client, nil
	}
	if err != repository.ErrNotFound {
		return nil, err
	}
	client = &models.Client{
		ID:                        uuid.New(),
		Name:                      name,
		Tier:                      tier,
		DefaultDestinationAddress: destination,
		CreatedAt:                 time.Now().UTC(),
	}
	if err := svc.repo.CreateClient(ctx, client); err != nil {
		return nil, err
	}
	return client, nil
}

func (svc *ClientService) GetClient(ctx context.Context, id uuid.UUID) (*models.Client, error) {
	if id == uuid.Nil {
		return nil, fmt.Errorf("%w: client id is required", middlewares.ErrValidation)
	}
	return svc.repo.GetClient(ctx, id)
}

func (svc *ClientService) ListClients(ctx context.Context) ([]models.Client, error) {
	clients, err := svc.repo.ListClients(ctx)
	if err != nil {
		return nil, err
	}
	if clients == nil {
		return []models.Client{}, nil
	}
	return clients, nil
}

func (svc *ClientService) UpdateClient(ctx context.Context, id uuid.UUID, req models.UpdateClientRequest) (*models.Client, error) {
	if id == uuid.Nil {
		return nil, fmt.Errorf("%w: client id is required", middlewares.ErrValidation)
	}
	client, err := svc.repo.GetClient(ctx, id)
	if err != nil {
		return nil, err
	}
	if req.Name == nil && req.Tier == nil && req.DefaultDestinationAddress == nil {
		return nil, fmt.Errorf("%w: update request is empty", middlewares.ErrValidation)
	}
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			return nil, fmt.Errorf("%w: client name cannot be empty", middlewares.ErrValidation)
		}
		client.Name = name
	}
	if req.Tier != nil {
		client.Tier = *req.Tier
	}
	if req.DefaultDestinationAddress != nil {
		client.DefaultDestinationAddress = strings.TrimSpace(*req.DefaultDestinationAddress)
	}
	if err := svc.repo.UpdateClient(ctx, client); err != nil {
		return nil, err
	}
	if svc.cache != nil {
		if err := svc.cache.ClearQueue(ctx); err != nil {
			// Client data is already saved; cache cleanup is optional.
		}
	}
	return client, nil
}
