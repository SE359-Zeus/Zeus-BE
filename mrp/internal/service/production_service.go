package service

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	"zeus-mrp-service/internal/infrastructure/messaging"
	"zeus-mrp-service/internal/models"

	"github.com/google/uuid"
)

func (s *ProductionService) PlanProduction(ctx context.Context, req models.CreateProductionOrderRequest) (*models.ProductionOrderResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// basic validation
	code := req.ProductModelCode
	if len(code) == 0 || len(code) > 128 {
		return nil, errors.New("product_model_code is required and must be reasonable length")
	}
	// disallow control/special characters — allow alnum, dash, underscore
	validCode := regexp.MustCompile(`^[A-Za-z0-9_\-]+$`)
	if !validCode.MatchString(code) {
		return nil, errors.New("product_model_code contains invalid characters")
	}
	if req.TargetQuantity <= 0 || req.TargetQuantity > 1_000_000 {
		return nil, errors.New("target_quantity must be >0 and <= 1000000")
	}

	// assign id and timestamps
	id := uuid.New()
	now := time.Now().UTC()

	order := &models.ProductionOrder{
		ID:               id,
		ProductModelCode: code,
		TargetQuantity:   req.TargetQuantity,
		Status:           models.StatusClearToBuild,
		ScheduledAt:      req.ScheduledAt,
		CreatedAt:        now,
	}

	if err := s.repo.CreateProductionOrder(ctx, order); err != nil {
		return nil, err
	}

	// Run BOM explosion to check readiness and detect shortages
	results, err := s.RunBOMExplosion(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to run BOM explosion: %w", err)
	}

	var shortageLogs []models.ShortageLog
	status := deriveReadinessStatus(results)
	if status != models.StatusClearToBuild {
		for _, result := range results {
			if result.IsShortage {
				shortageQty := result.TotalRequiredQty - result.AvailableQty
				// get human-readable component SKU
				var partSKU string
				if s.scmClient != nil {
					part, err := s.scmClient.GetPartCatalogByID(ctx, result.PartID)
					if err == nil && part != nil {
						partSKU = part.SKU
					}
				}
				if partSKU == "" {
					partSKU = fmt.Sprintf("PART-%s", result.PartID.String()[:8])
				}

				shortageLog := &models.ShortageLog{
					ID:                uuid.New(),
					ProductionOrderID: id,
					PartID:            result.PartID,
					ShortageQty:       shortageQty,
					ResolutionStatus:  models.ResolutionStatusShortage,
				}
				if err := s.repo.CreateShortageLog(ctx, shortageLog); err != nil {
					return nil, fmt.Errorf("failed to create shortage log: %w", err)
				}
				shortageLogs = append(shortageLogs, *shortageLog)

				// Emit shortage demand (deficit) to SCM
				if s.audit != nil {
					_ = s.audit.PublishJSON(ctx, messaging.DeficitPoolQueue, map[string]any{
						"sku":      partSKU,
						"qty":      shortageQty,
						"order_id": id.String(),
					})
				}
			}
		}

		// Update order status in DB
		if err := s.repo.UpdateProductionOrderStatus(ctx, id, status); err != nil {
			return nil, fmt.Errorf("failed to update production order status: %w", err)
		}
		order.Status = status
	}

	if shortageLogs == nil {
		shortageLogs = []models.ShortageLog{}
	}

	resp := &models.ProductionOrderResponse{
		ID:               order.ID,
		ProductModelCode: order.ProductModelCode,
		TargetQuantity:   order.TargetQuantity,
		Status:           order.Status,
		Shortages:        shortageLogs,
	}
	s.publishAudit(ctx, "CREATE", "mrp/production-orders/"+order.ID.String(), "Created production order for model "+order.ProductModelCode)
	return resp, nil
}
