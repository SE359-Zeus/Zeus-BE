package service

import (
	"context"
	"math"

	"zeus-scm-service/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type IVendorService interface {
	GetOptimalSupplier(ctx context.Context, sku string) (*models.Supplier, *models.SkuMapping, error)
	UpdateSupplierMetrics(ctx context.Context, supplierID uuid.UUID) error
}

type vendorService struct {
	db *gorm.DB
}

func NewVendorService(db *gorm.DB) IVendorService {
	return &vendorService{db: db}
}

func (s *vendorService) GetOptimalSupplier(ctx context.Context, sku string) (*models.Supplier, *models.SkuMapping, error) {
	var mappings []models.SkuMapping
	if err := s.db.WithContext(ctx).
		Preload("Supplier").
		Where("sku = ?", sku).
		Order("unit_price ASC").
		Find(&mappings).Error; err != nil {
		return nil, nil, err
	}
	if len(mappings) == 0 {
		return nil, nil, ErrNoOptimalSupplier
	}
	var bestSupplier *models.Supplier
	var bestMapping *models.SkuMapping
	bestScore := -1.0
	for i := range mappings {
		supplier := &models.Supplier{}
		if err := s.db.WithContext(ctx).First(supplier, "id = ?", mappings[i].SupplierID).Error; err != nil {
			continue
		}
		score := (supplier.QualityScore*0.6 + supplier.OnTimeRate*0.4) - (mappings[i].UnitPrice / 10000.0)
		if score > bestScore {
			bestScore = score
			bestSupplier = supplier
			bestMapping = &mappings[i]
		}
	}
	if bestSupplier == nil {
		return nil, nil, ErrNoOptimalSupplier
	}
	return bestSupplier, bestMapping, nil
}

func (s *vendorService) UpdateSupplierMetrics(ctx context.Context, supplierID uuid.UUID) error {
	var totalGRs int64
	var defectiveGRs int64
	var onTimeGRs int64

	s.db.WithContext(ctx).Model(&models.GoodsReceipt{}).
		Where("vendor_id = ?", supplierID).
		Count(&totalGRs)

	if totalGRs == 0 {
		s.db.WithContext(ctx).Model(&models.Supplier{}).
			Where("id = ?", supplierID).
			Updates(map[string]interface{}{
				"on_time_rate":  0,
				"quality_score": 100,
				"updated_at":    nil,
			})
		return nil
	}

	var receipts []models.GoodsReceipt
	s.db.WithContext(ctx).Where("vendor_id = ?", supplierID).Find(&receipts)
	for _, gr := range receipts {
		var items []models.GRLineItem
		s.db.WithContext(ctx).Where("gr_id = ?", gr.ID).Find(&items)
		for _, item := range items {
			if item.DefectiveQty != nil && *item.DefectiveQty > 0 {
				defectiveGRs++
			}
		}
		if gr.Status == models.GRStatusComplete {
			onTimeGRs++
		}
	}

	onTimeRate := float64(onTimeGRs) / math.Max(float64(totalGRs), 1) * 100
	qualityScore := 100.0 - (float64(defectiveGRs)/math.Max(float64(totalGRs), 1))*100

	return s.db.WithContext(ctx).Model(&models.Supplier{}).
		Where("id = ?", supplierID).
		Updates(map[string]interface{}{
			"on_time_rate":  math.Round(onTimeRate*100) / 100,
			"quality_score": math.Round(qualityScore*100) / 100,
		}).Error
}
