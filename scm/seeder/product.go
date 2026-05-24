package seeder

import (
	"fmt"
	"time"

	"zeus-scm-service/internal/models"

	"github.com/brianvoe/gofakeit/v6"
	"gorm.io/gorm"
)

func seedProductsAndParts(db *gorm.DB, modelsList []models.ProductModel, catMap map[string]models.PartCatalog) {
	_ = catMap
	for modelIndex, pm := range modelsList {
		for productIndex := 0; productIndex < 2; productIndex++ {
			prod := models.Product{
				ID:               stableUUID("product:" + pm.ModelCode + ":" + fmt.Sprintf("%d", productIndex)),
				ProductModelCode: pm.ModelCode,
				CustomerID:       stableUUID("customer:" + pm.ModelCode + ":" + fmt.Sprintf("%d", productIndex)),
				ProductName:      pm.ModelName + " Build " + gofakeit.LetterN(3),
				SerialNumber:     fmt.Sprintf("SN-%s-%02d", pm.ModelCode, productIndex+1),
				CreatedAt:        time.Now(),
				UpdatedAt:        time.Now(),
			}
			db.Where("id = ?", prod.ID).Assign(prod).FirstOrCreate(&prod)

			var boms []models.PartsByModel
			db.Where("product_model_code = ?", pm.ModelCode).Find(&boms)

			for _, bom := range boms {
				for q := int32(0); q < bom.Quantity; q++ {
					pid := prod.ID
					p := models.Part{
						ID:               stableUUID("part:" + prod.ID.String() + ":" + bom.PartCatalogID.String() + ":" + fmt.Sprintf("%d", q)),
						PartCatalogID:    bom.PartCatalogID,
						ProductID:        &pid,
						SerialNumber:     fmt.Sprintf("PART-%s-%d-%d", pm.ModelCode, modelIndex, q+1),
						PartConditionID:  1,
						ManufacturedDate: time.Now().AddDate(0, -gofakeit.Number(1, 12), 0),
						InstallationDate: &prod.CreatedAt,
						CreatedAt:        time.Now(),
						UpdatedAt:        time.Now(),
					}
					db.Where("id = ?", p.ID).Assign(p).FirstOrCreate(&p)
				}
			}
		}
	}
}
