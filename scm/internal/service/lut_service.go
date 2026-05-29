package service

import (
	"context"

	"zeus-scm-service/internal/models"
	"zeus-scm-service/internal/repository"
)

type ILUTService interface {
	GetAllLUTs(ctx context.Context) (*LUTCollection, error)
}

type LUTCollection struct {
	PartTypes           []models.PartType           `json:"part_types"`
	PartConditions      []models.PartCondition      `json:"part_conditions"`
	PartMfgStatuses     []models.PartMfgStatus      `json:"part_mfg_statuses"`
	ComponentStockStates []models.ComponentStockState `json:"component_stock_states"`
	PurchaseOrderStates []models.PurchaseOrderState `json:"purchase_order_states"`
	GoodsReceiptStates  []models.GoodsReceiptState  `json:"goods_receipt_states"`
	ShipmentStates      []models.ShipmentState      `json:"shipment_states"`
}

type lutService struct {
	repo repository.ILUTRepository
}

func NewLUTService(repo repository.ILUTRepository) ILUTService {
	return &lutService{repo: repo}
}

func (s *lutService) GetAllLUTs(ctx context.Context) (*LUTCollection, error) {
	partTypes, err := s.repo.ListPartTypes(ctx)
	if err != nil {
		return nil, err
	}
	partConditions, err := s.repo.ListPartConditions(ctx)
	if err != nil {
		return nil, err
	}
	partMfgStatuses, err := s.repo.ListPartMfgStatuses(ctx)
	if err != nil {
		return nil, err
	}
	componentStockStates, err := s.repo.ListComponentStockStates(ctx)
	if err != nil {
		return nil, err
	}
	purchaseOrderStates, err := s.repo.ListPurchaseOrderStates(ctx)
	if err != nil {
		return nil, err
	}
	goodsReceiptStates, err := s.repo.ListGoodsReceiptStates(ctx)
	if err != nil {
		return nil, err
	}
	shipmentStates, err := s.repo.ListShipmentStates(ctx)
	if err != nil {
		return nil, err
	}

	return &LUTCollection{
		PartTypes:           partTypes,
		PartConditions:      partConditions,
		PartMfgStatuses:     partMfgStatuses,
		ComponentStockStates: componentStockStates,
		PurchaseOrderStates: purchaseOrderStates,
		GoodsReceiptStates:  goodsReceiptStates,
		ShipmentStates:      shipmentStates,
	}, nil
}
