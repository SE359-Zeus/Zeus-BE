package service

import (
	"context"
	"math"
	"strings"

	"zeus-scm-service/internal/models"
	"zeus-scm-service/internal/pagination"
	"zeus-scm-service/internal/repository"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type IVendorService interface {
	GetOptimalSupplier(ctx context.Context, sku string) (*models.Supplier, *models.SkuMapping, error)
	UpdateSupplierMetrics(ctx context.Context, supplierID uuid.UUID) error
	ListSuppliers(ctx context.Context, tier string, params pagination.Params, q string) ([]models.Supplier, *pagination.Meta, error)
	CreateSupplier(ctx context.Context, supplier *models.Supplier) error
	CreateSkuMapping(ctx context.Context, mapping *models.SkuMapping) error
	GetSupplierMetrics(ctx context.Context) (int64, float64, error)
	FindAllSuppliersWithMappings(ctx context.Context) ([]models.Supplier, error)
}

type vendorService struct {
	db *gorm.DB
}

type vendorServiceRepo struct {
	repo repository.IVendorRepository
}

func NewVendorService(arg interface{}) IVendorService {
	switch v := arg.(type) {
	case *gorm.DB:
		return &vendorService{db: v}
	case repository.IVendorRepository:
		return &vendorServiceRepo{repo: v}
	default:
		panic("invalid NewVendorService usage")
	}
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

// repo-backed implementation
func (s *vendorServiceRepo) GetOptimalSupplier(ctx context.Context, sku string) (*models.Supplier, *models.SkuMapping, error) {
	mappings, err := s.repo.FindSkuMappingsBySKU(ctx, sku)
	if err != nil || len(mappings) == 0 {
		return nil, nil, ErrNoOptimalSupplier
	}
	var bestSupplier *models.Supplier
	var bestMapping *models.SkuMapping
	bestScore := -1.0
	for i := range mappings {
		supplier, err := s.repo.GetSupplierByID(ctx, mappings[i].SupplierID)
		if err != nil || supplier == nil {
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

func (s *vendorServiceRepo) UpdateSupplierMetrics(ctx context.Context, supplierID uuid.UUID) error {
	totalGRs, err := s.repo.CountGoodsReceiptsByVendor(ctx, supplierID)
	if err != nil {
		return err
	}
	if totalGRs == 0 {
		return s.repo.UpdateSupplier(ctx, supplierID, map[string]interface{}{
			"on_time_rate":  0,
			"quality_score": 100,
			"updated_at":    nil,
		})
	}

	receipts, err := s.repo.FindGoodsReceiptsByVendor(ctx, supplierID)
	if err != nil {
		return err
	}

	var defectiveGRs int64
	var onTimeGRs int64
	for _, gr := range receipts {
		items, _ := s.repo.FindGRLineItemsByGRID(ctx, gr.ID)
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

	return s.repo.UpdateSupplier(ctx, supplierID, map[string]interface{}{
		"on_time_rate":  math.Round(onTimeRate*100) / 100,
		"quality_score": math.Round(qualityScore*100) / 100,
	})
}

func (s *vendorService) ListSuppliers(ctx context.Context, tier string, params pagination.Params, q string) ([]models.Supplier, *pagination.Meta, error) {
	query := s.db.WithContext(ctx).Model(&models.Supplier{}).Preload("SkuMappings")
	if tier != "" {
		switch strings.ToLower(tier) {
		case "tier 1", "tier1":
			query = query.Where("tier = ?", models.SupplierTier1)
		case "tier 2", "tier2":
			query = query.Where("tier = ?", models.SupplierTier2)
		case "tier 3", "tier3":
			query = query.Where("tier = ?", models.SupplierTier3)
		default:
			query = query.Where("tier = ?", tier)
		}
	}
	if q != "" {
		like := "%" + q + "%"
		query = query.Where("name LIKE ? OR contact LIKE ? OR id LIKE ?", like, like, like)
	}
	var suppliers []models.Supplier
	meta, err := pagination.Paginate(query, params, &suppliers, "created_at", "updated_at", "name", "contact")
	if err != nil {
		return nil, nil, err
	}
	return suppliers, meta, nil
}

func (s *vendorService) CreateSupplier(ctx context.Context, supplier *models.Supplier) error {
	return s.db.WithContext(ctx).Create(supplier).Error
}

func (s *vendorService) CreateSkuMapping(ctx context.Context, mapping *models.SkuMapping) error {
	return s.db.WithContext(ctx).Create(mapping).Error
}

func (s *vendorService) GetSupplierMetrics(ctx context.Context) (int64, float64, error) {
	var count int64
	if err := s.db.WithContext(ctx).Model(&models.Supplier{}).Count(&count).Error; err != nil {
		return 0, 0, err
	}
	var avg float64
	err := s.db.WithContext(ctx).Model(&models.Supplier{}).Select("COALESCE(AVG(on_time_rate), 0)").Row().Scan(&avg)
	if err != nil {
		return 0, 0, err
	}
	return count, avg, nil
}

func (s *vendorService) FindAllSuppliersWithMappings(ctx context.Context) ([]models.Supplier, error) {
	var suppliers []models.Supplier
	if err := s.db.WithContext(ctx).Preload("SkuMappings").Order("name ASC").Find(&suppliers).Error; err != nil {
		return nil, err
	}
	return suppliers, nil
}

func (s *vendorServiceRepo) ListSuppliers(ctx context.Context, tier string, params pagination.Params, q string) ([]models.Supplier, *pagination.Meta, error) {
	return s.repo.ListSuppliers(ctx, tier, params, q)
}

func (s *vendorServiceRepo) CreateSupplier(ctx context.Context, supplier *models.Supplier) error {
	return s.repo.CreateSupplier(ctx, supplier)
}

func (s *vendorServiceRepo) CreateSkuMapping(ctx context.Context, mapping *models.SkuMapping) error {
	return s.repo.CreateSkuMapping(ctx, mapping)
}

func (s *vendorServiceRepo) GetSupplierMetrics(ctx context.Context) (int64, float64, error) {
	count, err := s.repo.CountSuppliers(ctx)
	if err != nil {
		return 0, 0, err
	}
	avg, err := s.repo.GetAverageOnTimeRate(ctx)
	if err != nil {
		return 0, 0, err
	}
	return count, avg, nil
}

func (s *vendorServiceRepo) FindAllSuppliersWithMappings(ctx context.Context) ([]models.Supplier, error) {
	return s.repo.FindAllSuppliersWithMappings(ctx)
}
