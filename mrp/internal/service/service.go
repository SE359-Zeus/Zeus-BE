package service

import (
	"context"

	"github.com/google/uuid"

	"zeus-mrp-service/internal/infrastructure/messaging"
	"zeus-mrp-service/internal/middlewares"
	"zeus-mrp-service/internal/models"
	"zeus-mrp-service/internal/repository"
)

type AuditPublisher interface {
	PublishJSON(ctx context.Context, queue string, payload any) error
}

type SCMClient interface {
	GetPartCatalogBySKU(ctx context.Context, sku string) (*models.Part, error)
	GetPartCatalogByID(ctx context.Context, id uuid.UUID) (*models.Part, error)
	GetStockBySKU(ctx context.Context, sku string) (*models.ComponentStock, error)
	GetProductModelByCode(ctx context.Context, code string) (*models.ProductModel, error)
	ListStocks(ctx context.Context, page, limit int, sortBy, sortDir, q string) ([]models.ComponentStock, bool, error)
	CreateCatalogPart(ctx context.Context, sku, description string, price float64) (*models.Part, error)
	UpdateCatalogPart(ctx context.Context, sku, description string, price float64) (*models.Part, error)
	DeleteCatalogPart(ctx context.Context, sku string) error
	GetInventoryLedger(ctx context.Context, page, limit int, sortBy, sortDir, txnType, sku string) ([]models.InventoryLedgerEntry, bool, error)
	GetLUTs(ctx context.Context) (*models.LUTCollection, error)
}

type ProductionService struct {
	repo      repository.MRPRepository
	cache     repository.CacheRepository
	scmClient SCMClient
	audit     AuditPublisher
}

func NewProductionService(repo repository.MRPRepository, deps ...any) *ProductionService {
	svc := &ProductionService{repo: repo}
	for _, dep := range deps {
		switch d := dep.(type) {
		case SCMClient:
			svc.scmClient = d
		case repository.CacheRepository:
			svc.cache = d
		case AuditPublisher:
			svc.audit = d
		}
	}
	return svc
}

func (s *ProductionService) publishAudit(ctx context.Context, actionType, targetResource, details string) {
	if s == nil || s.audit == nil {
		return
	}
	userID, _ := ctx.Value(middlewares.ContextKeyUserID).(string)
	userEmail, _ := ctx.Value(middlewares.ContextKeyEmail).(string)
	if userID == "" || userEmail == "" {
		return
	}
	_ = s.audit.PublishJSON(ctx, messaging.AuditQueue, map[string]any{
		"user_id":         userID,
		"user_email":      userEmail,
		"action_type":     normalizeAuditActionType(actionType),
		"target_resource": targetResource,
		"details":         details,
	})
}
