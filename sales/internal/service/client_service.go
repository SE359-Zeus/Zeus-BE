package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	infraMessaging "zeus-sales-service/internal/infrastructure/messaging"
	"zeus-sales-service/internal/infrastructure/observability"
	"zeus-sales-service/internal/middlewares"
	"zeus-sales-service/internal/models"
	"zeus-sales-service/internal/repository"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type ClientService struct {
	repo  repository.DbRepository
	cache repository.CacheRepository
	infra *Infrastructure
}

func NewClientService(repo repository.DbRepository, cache repository.CacheRepository, infra ...*Infrastructure) *ClientService {
	var sharedInfra *Infrastructure
	if len(infra) > 0 {
		sharedInfra = infra[0]
	}
	return &ClientService{repo: repo, cache: cache, infra: sharedInfra}
}

func (svc *ClientService) ResolveOrCreateClient(ctx context.Context, name string, destination string, tier models.ClientTier) (*models.Client, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("%w: client name is required", middlewares.ErrValidation)
	}
	if cached := svc.getCachedClientByName(ctx, name); cached != nil {
		return cached, nil
	}
	if tier == "" {
		tier = models.ClientTierB2C
	}
	exists, err := svc.repo.ExistsClientByName(ctx, name)
	if err != nil {
		return nil, err
	}
	if exists {
		client, err := svc.repo.GetClientByName(ctx, name)
		if err != nil {
			return nil, err
		}
		svc.cacheClient(ctx, client)
		return client, nil
	}
	client := &models.Client{
		ID:                        uuid.New(),
		Name:                      name,
		Tier:                      tier,
		DefaultDestinationAddress: destination,
		CreatedAt:                 time.Now().UTC(),
	}
	if err := svc.repo.CreateClient(ctx, client); err != nil {
		return nil, err
	}
	svc.cacheClient(ctx, client)
	return client, nil
}

func (svc *ClientService) CreateClient(ctx context.Context, req models.CreateClientRequest) (*models.Client, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, fmt.Errorf("%w: client name is required", middlewares.ErrValidation)
	}
	if req.Tier != models.ClientTierB2B && req.Tier != models.ClientTierB2C {
		return nil, fmt.Errorf("%w: tier must be B2B or B2C", middlewares.ErrValidation)
	}

	// Reject duplicates — name is unique in the clients table
	exists, err := svc.repo.ExistsClientByName(ctx, name)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, fmt.Errorf("%w: a client with this name already exists", middlewares.ErrConflict)
	}

	client := &models.Client{
		ID:                        uuid.New(),
		Name:                      name,
		Tier:                      req.Tier,
		DefaultDestinationAddress: strings.TrimSpace(req.DefaultDestinationAddress),
		CreatedAt:                 time.Now().UTC(),
		UpdatedAt:                 time.Now().UTC(),
	}
	if err := svc.repo.CreateClient(ctx, client); err != nil {
		return nil, err
	}
	svc.cacheClient(ctx, client)
	if svc.infra != nil && svc.infra.Publisher != nil {
		_ = svc.infra.Publisher.Publish(ctx, "sales.client.created", client)
		svc.publishAudit(ctx, "CREATE", "sales/clients/"+client.ID.String(), "Created client "+client.Name)
	}
	observability.DefaultRegistry.Counter(observability.MetricClientsCreated).Inc()
	return client, nil
}

func (svc *ClientService) GetClient(ctx context.Context, id uuid.UUID) (*models.Client, error) {
	if id == uuid.Nil {
		return nil, fmt.Errorf("%w: client id is required", middlewares.ErrValidation)
	}
	if cached := svc.getCachedClientByID(ctx, id); cached != nil {
		return cached, nil
	}
	client, err := svc.repo.GetClient(ctx, id)
	if err != nil {
		return nil, err
	}
	svc.cacheClient(ctx, client)
	return client, nil
}

func (svc *ClientService) ListClients(ctx context.Context) ([]models.Client, error) {
	if cached := svc.getCachedClients(ctx); cached != nil {
		return cached, nil
	}
	clients, err := svc.repo.ListClients(ctx)
	if err != nil {
		return nil, err
	}
	if clients == nil {
		return []models.Client{}, nil
	}
	svc.cacheClients(ctx, clients)
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
	originalName := client.Name
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
	if originalName != client.Name && svc.infra != nil && svc.infra.Cache != nil {
		_ = svc.infra.Cache.DeleteClient(ctx, client.ID, originalName)
	}
	svc.cacheClient(ctx, client)
	if svc.cache != nil {
		if err := svc.cache.ClearQueue(ctx); err != nil {
			// Client data is already saved; queue cache cleanup is optional.
		}
	}
	if svc.infra != nil && svc.infra.Publisher != nil {
		_ = svc.infra.Publisher.Publish(ctx, "sales.client.updated", client)
		svc.publishAudit(ctx, "UPDATE", "sales/clients/"+client.ID.String(), "Updated client "+client.Name)
	}
	return client, nil
}

func (svc *ClientService) DeleteClient(ctx context.Context, id uuid.UUID) error {
	if id == uuid.Nil {
		return fmt.Errorf("%w: client id is required", middlewares.ErrValidation)
	}
	client, err := svc.repo.GetClient(ctx, id)
	if err != nil {
		return err
	}
	if err := svc.repo.DeleteClient(ctx, id); err != nil {
		return err
	}
	if svc.infra != nil && svc.infra.Cache != nil && client != nil {
		_ = svc.infra.Cache.DeleteClient(ctx, client.ID, client.Name)
	}
	if svc.infra != nil && svc.infra.Publisher != nil && client != nil {
		svc.publishAudit(ctx, "DELETE", "sales/clients/"+client.ID.String(), "Deleted client "+client.Name)
	}
	return nil
}

func (svc *ClientService) cacheClient(ctx context.Context, client *models.Client) {
	if svc == nil || svc.infra == nil || svc.infra.Cache == nil || client == nil {
		return
	}
	_ = svc.infra.Cache.SetClient(ctx, *client)
}

func (svc *ClientService) cacheClients(ctx context.Context, clients []models.Client) {
	if svc == nil || svc.infra == nil || svc.infra.Cache == nil {
		return
	}
	_ = svc.infra.Cache.SetClients(ctx, clients)
}

func (svc *ClientService) getCachedClientByID(ctx context.Context, id uuid.UUID) *models.Client {
	if svc == nil || svc.infra == nil || svc.infra.Cache == nil {
		return nil
	}
	client, ok, err := svc.infra.Cache.GetClientByID(ctx, id)
	if err != nil || !ok {
		return nil
	}
	return client
}

func (svc *ClientService) getCachedClientByName(ctx context.Context, name string) *models.Client {
	if svc == nil || svc.infra == nil || svc.infra.Cache == nil {
		return nil
	}
	client, ok, err := svc.infra.Cache.GetClientByName(ctx, name)
	if err != nil || !ok {
		return nil
	}
	return client
}

func (svc *ClientService) getCachedClients(ctx context.Context) []models.Client {
	if svc == nil || svc.infra == nil || svc.infra.Cache == nil {
		return nil
	}
	clients, ok, err := svc.infra.Cache.GetClients(ctx)
	if err != nil || !ok {
		return nil
	}
	return clients
}

func (svc *ClientService) publishAudit(ctx context.Context, actionType string, targetResource string, details string) {
	if svc == nil || svc.infra == nil || svc.infra.Publisher == nil {
		return
	}
	userID, _ := ctx.Value(middlewares.ContextKeyUserID).(string)
	email, _ := ctx.Value(middlewares.ContextKeyEmail).(string)
	if strings.TrimSpace(userID) == "" || strings.TrimSpace(email) == "" {
		return
	}
	_ = svc.infra.Publisher.Publish(ctx, infraMessaging.AuditQueue, map[string]any{
		"user_id":         userID,
		"user_email":      email,
		"action_type":     strings.ToUpper(strings.TrimSpace(actionType)),
		"target_resource": targetResource,
		"details":         details,
	})
}

func (svc *ClientService) VerifyClientAPIKey(ctx context.Context, prefix, rawKey string) (*models.Client, error) {
	client, err := svc.repo.GetClientByAPIKeyPrefix(ctx, prefix)
	if err != nil {
		return nil, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(client.ApiKeyHash), []byte(rawKey)); err != nil {
		return nil, fmt.Errorf("invalid api key: %w", err)
	}
	return client, nil
}

