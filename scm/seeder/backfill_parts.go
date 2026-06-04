package seeder

import (
	"fmt"
	"log/slog"
	"time"

	"zeus-scm-service/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// BackfillPartsForEmptyProducts finds products that have zero parts and
// generates parts from the BOM (parts_by_model) for their product model.
// It also cleans up products with blank product_model_code or serial_number.
func BackfillPartsForEmptyProducts(db *gorm.DB) error {
	slog.Info("backfill: starting",
		slog.String("service", "scm"),
		slog.String("event", "backfill_start"),
	)

	// Step 1: Delete products with empty product_model_code or serial_number.
	// These are corrupt rows (e.g. failed inserts, partial data).
	var deletedCount int64
	result := db.Where("product_model_code = '' OR product_model_code IS NULL OR serial_number = '' OR serial_number IS NULL").
		Delete(&models.Product{})
	if result.Error != nil {
		return fmt.Errorf("backfill: delete corrupt products: %w", result.Error)
	}
	deletedCount = result.RowsAffected
	if deletedCount > 0 {
		slog.Info("backfill: deleted corrupt products",
			slog.String("service", "scm"),
			slog.String("event", "backfill_deleted_corrupt"),
			slog.Int64("count", deletedCount),
		)
	}

	// Step 2: Find all products.
	var products []models.Product
	if err := db.Find(&products).Error; err != nil {
		return fmt.Errorf("backfill: list products: %w", err)
	}

	if len(products) == 0 {
		slog.Info("backfill: no products found, nothing to do",
			slog.String("service", "scm"),
			slog.String("event", "backfill_done"),
		)
		return nil
	}

	// Step 3: For each product, check if it has parts. If not, generate them.
	var backfilled, skipped, failed int
	for i := range products {
		prod := &products[i]

		// Validate product has a model code.
		if prod.ProductModelCode == "" {
			slog.Warn("backfill: skipping product with empty model code",
				slog.String("service", "scm"),
				slog.String("product_id", prod.ID.String()),
			)
			skipped++
			continue
		}

		// Check if product already has parts.
		var partCount int64
		if err := db.Model(&models.Part{}).Where("product_id = ?", prod.ID).Count(&partCount).Error; err != nil {
			slog.Warn("backfill: failed to count parts",
				slog.String("service", "scm"),
				slog.String("product_id", prod.ID.String()),
				slog.Any("error", err),
			)
			failed++
			continue
		}
		if partCount > 0 {
			skipped++
			continue
		}

		// Look up BOM for this product's model.
		var boms []models.PartsByModel
		if err := db.Where("product_model_code = ?", prod.ProductModelCode).Find(&boms).Error; err != nil {
			slog.Warn("backfill: failed to load BOM",
				slog.String("service", "scm"),
				slog.String("product_id", prod.ID.String()),
				slog.String("model_code", prod.ProductModelCode),
				slog.Any("error", err),
			)
			failed++
			continue
		}
		if len(boms) == 0 {
			slog.Warn("backfill: no BOM entries for model, skipping",
				slog.String("service", "scm"),
				slog.String("product_id", prod.ID.String()),
				slog.String("model_code", prod.ProductModelCode),
			)
			skipped++
			continue
		}

		// Generate parts from BOM.
		now := time.Now()
		partsCreated := 0
		for _, bom := range boms {
			for q := int32(0); q < bom.Quantity; q++ {
				part := models.Part{
					ID:               uuid.New(),
					PartCatalogID:    bom.PartCatalogID,
					ProductID:        &prod.ID,
					SerialNumber:     fmt.Sprintf("PART-%s-%d", prod.ID.String()[:8], q+1),
					PartConditionID:  1,
					ManufacturedDate: now,
					InstallationDate: &prod.CreatedAt,
					CreatedAt:        now,
					UpdatedAt:        now,
				}
				if err := db.Create(&part).Error; err != nil {
					slog.Warn("backfill: failed to create part",
						slog.String("service", "scm"),
						slog.String("product_id", prod.ID.String()),
						slog.Any("error", err),
					)
					failed++
					continue
				}
				partsCreated++
			}
		}

		slog.Info("backfill: created parts for product",
			slog.String("service", "scm"),
			slog.String("product_id", prod.ID.String()),
			slog.String("model_code", prod.ProductModelCode),
			slog.Int("parts_created", partsCreated),
		)
		backfilled++
	}

	slog.Info("backfill: finished",
		slog.String("service", "scm"),
		slog.String("event", "backfill_done"),
		slog.Int("backfilled", backfilled),
		slog.Int("skipped", skipped),
		slog.Int("failed", failed),
		slog.Int64("corrupt_deleted", deletedCount),
	)
	return nil
}
