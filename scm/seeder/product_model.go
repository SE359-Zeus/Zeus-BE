package seeder

import (
	"fmt"
	"log"
	"zeus-scm-service/internal/models"

	"time"

	"gorm.io/gorm"
)

func seedProductModels(db *gorm.DB, installs map[string][]PartInstallationData, catMap map[string]models.PartCatalog) []models.ProductModel {
	if err := ensurePartsByModelTable(db); err != nil {
		log.Printf("warning: failed to ensure parts_by_model table: %v", err)
	}

	var existingCount int64
	if err := db.Model(&models.ProductModel{}).Count(&existingCount).Error; err == nil && existingCount > 0 {
		var modelsList []models.ProductModel
		db.Find(&modelsList)
		return modelsList
	}

	baseModels := []models.ProductModel{
		{ModelCode: "82SN003JVN", ModelName: "IdeaPad 5 Pro 16ARH7"},
		{ModelCode: "83LY00HQVN", ModelName: "Legion 5 15IRX10"},
	}

	newModels := []models.ProductModel{
		{ModelCode: "21CB000QUS", ModelName: "ThinkPad X1 Carbon Gen 11"},
		{ModelCode: "82A3000GUS", ModelName: "Yoga Slim 7i"},
		{ModelCode: "82WQ002RUS", ModelName: "Legion Pro 7i"},
	}

	allModels := append(baseModels, newModels...)

	for _, m := range allModels {
		desc := "Seeded product model"
		m.Description = &desc
		m.CreatedAt = time.Now()
		m.UpdatedAt = time.Now()
		db.Where("model_code = ?", m.ModelCode).Assign(m).FirstOrCreate(&m)

		bomList, exists := installs[m.ModelCode]
		if !exists {
			bomList = installs["82SN003JVN"]
		}

		for _, item := range bomList {
			key := fmt.Sprintf("%s|%s", item.PartNumber, item.MfgNumber)
			if cat, ok := catMap[key]; ok {
				bom := models.PartsByModel{
					PartCatalogID:    cat.ID,
					ProductModelCode: m.ModelCode,
					Quantity:         int32(item.Quantity),
				}
				db.Where("part_catalog_id = ? AND product_model_code = ?", cat.ID, m.ModelCode).Assign(bom).FirstOrCreate(&bom)
			}
		}
	}
	return allModels
}

func ensurePartsByModelTable(db *gorm.DB) error {
	return db.Exec(`CREATE TABLE IF NOT EXISTS parts_by_model (
    part_catalog_id TEXT NOT NULL,
    product_model_code TEXT NOT NULL,
    quantity INTEGER NOT NULL,
    image_name TEXT,
    PRIMARY KEY (part_catalog_id, product_model_code)
)`).Error
}
