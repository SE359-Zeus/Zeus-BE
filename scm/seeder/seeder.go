package seeder

import (
	"fmt"
	"log/slog"

	"github.com/brianvoe/gofakeit/v6"
	"gorm.io/gorm"
)

func SeedAll(db *gorm.DB, partsDataPath string, manifestOutPath string) error {
	slog.Info("starting scm seeder", slog.String("service", "scm"), slog.String("event", "seed_start"))
	gofakeit.Seed(0)

	seedLookupTables(db)
	_ = seedAPIKeys(db)
	slog.Info("seeded scm api key",
		slog.String("service", "scm"),
		slog.String("event", "seeded_api_key"),
		slog.String("api_key_name", defaultAPIKeyName),
		slog.String("api_key_prefix", defaultAPIKeyPrefix),
	)
	suppliers := seedSuppliers(db, 5)

	data, err := loadPartsData(partsDataPath)
	if err != nil {
		return fmt.Errorf("failed to load parts data: %w", err)
	}

	typeMap, catMap := seedCatalogs(db, data)
	_ = typeMap
	modelsList := seedProductModels(db, data.Installations, catMap)
	seedInventory(db, catMap, suppliers)
	seedProcurementData(db, suppliers, data)
	seedProductsAndParts(db, modelsList, catMap)

	if err := writeSeedManifest(db, manifestOutPath); err != nil {
		slog.Warn("failed to write seed manifest",
			slog.String("service", "scm"),
			slog.String("event", "manifest_write_failed"),
			slog.String("path", manifestOutPath),
			slog.Any("error", err),
		)
	}

	slog.Info("scm seeder finished successfully", slog.String("service", "scm"), slog.String("event", "seed_complete"))
	return nil
}
