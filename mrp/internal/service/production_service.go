package service

import (
	"context"
	"errors"
	"regexp"
	"time"
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

	resp := &models.ProductionOrderResponse{
		ID:               order.ID,
		ProductModelCode: order.ProductModelCode,
		TargetQuantity:   order.TargetQuantity,
		Status:           order.Status,
		Shortages:        []models.ShortageLog{},
	}
	s.publishAudit(ctx, "CREATE", "mrp/production-orders/"+order.ID.String(), "Created production order for model "+order.ProductModelCode)
	return resp, nil
}
