package seeder

import (
	"fmt"
	"log"

	"github.com/brianvoe/gofakeit/v6"
	"gorm.io/gorm"
)

func SeedAll(db *gorm.DB, partsDataPath string, manifestOutPath string) error {
	log.Println("Starting SCM Seeder...")
	gofakeit.Seed(0)

	seedLookupTables(db)
	apiKey := seedAPIKeys(db)
	log.Printf("Seeded SCM API key: %s", apiKey)
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
		return fmt.Errorf("failed to write seed manifest: %w", err)
	}

	log.Println("SCM Seeder finished successfully.")
	return nil
}
