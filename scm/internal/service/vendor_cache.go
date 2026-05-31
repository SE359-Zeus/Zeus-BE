package service

import (
	"context"
	"log/slog"

	"zeus-scm-service/internal/models"
	"zeus-scm-service/internal/pagination"
	"zeus-scm-service/internal/repository"

	"github.com/google/uuid"
)

type cachedVendorService struct {
	base        IVendorService
	vendorCache repository.IVendorCache
	repo        repository.IVendorRepository
}

func NewCachedVendorService(base IVendorService, vendorCache repository.IVendorCache, repo repository.IVendorRepository) IVendorService {
	return &cachedVendorService{
		base:        base,
		vendorCache: vendorCache,
		repo:        repo,
	}
}

func (s *cachedVendorService) GetOptimalSupplier(ctx context.Context, sku string) (*models.Supplier, *models.SkuMapping, error) {
	if s.vendorCache != nil {
		if supplier, mapping, err := s.vendorCache.GetOptimalSupplier(ctx, sku); err == nil && supplier != nil && mapping != nil {
			return supplier, mapping, nil
		}
	}
	supplier, mapping, err := s.base.GetOptimalSupplier(ctx, sku)
	if err != nil {
		return nil, nil, err
	}
	if s.vendorCache != nil {
		_ = s.vendorCache.SetOptimalSupplier(ctx, sku, supplier, mapping)
	}
	return supplier, mapping, nil
}

func (s *cachedVendorService) UpdateSupplierMetrics(ctx context.Context, supplierID uuid.UUID) error {
	// Call the base service first to update metrics in DB
	err := s.base.UpdateSupplierMetrics(ctx, supplierID)
	if err != nil {
		return err
	}

	// Invalidate the cache for all SKUs affected by this supplier
	if s.vendorCache != nil && s.repo != nil {
		mappings, err := s.repo.FindSkuMappingsBySupplierID(ctx, supplierID)
		if err == nil {
			for _, m := range mappings {
				_ = s.vendorCache.DeleteOptimalSupplier(ctx, m.SKU)
			}
		} else {
			slog.Warn("failed to find SKU mappings for supplier to invalidate cache",
				slog.String("service", "scm"),
				slog.String("component", "vendor_cache"),
				slog.String("supplier_id", supplierID.String()),
				slog.Any("error", err),
			)
		}
	}
	return nil
}

func (s *cachedVendorService) ListSuppliers(ctx context.Context, tier string, params pagination.Params, q string) ([]models.Supplier, *pagination.Meta, error) {
	return s.base.ListSuppliers(ctx, tier, params, q)
}

func (s *cachedVendorService) CreateSupplier(ctx context.Context, supplier *models.Supplier) error {
	return s.base.CreateSupplier(ctx, supplier)
}

func (s *cachedVendorService) CreateSkuMapping(ctx context.Context, mapping *models.SkuMapping) error {
	err := s.base.CreateSkuMapping(ctx, mapping)
	if err != nil {
		return err
	}
	if s.vendorCache != nil {
		_ = s.vendorCache.DeleteOptimalSupplier(ctx, mapping.SKU)
	}
	return nil
}

func (s *cachedVendorService) GetSupplierMetrics(ctx context.Context) (int64, float64, error) {
	return s.base.GetSupplierMetrics(ctx)
}

func (s *cachedVendorService) FindAllSuppliersWithMappings(ctx context.Context) ([]models.Supplier, error) {
	return s.base.FindAllSuppliersWithMappings(ctx)
}

func (s *cachedVendorService) GetShortageSummary(ctx context.Context) ([]models.ShortageSummaryDTO, error) {
	return s.base.GetShortageSummary(ctx)
}
