package service

import (
	"context"
	"log"

	"zeus-scm-service/internal/models"
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
			log.Printf("warning: failed to find SKU mappings for supplier %s to invalidate cache: %v", supplierID.String(), err)
		}
	}
	return nil
}
