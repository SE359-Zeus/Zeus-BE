package seeder

import (
	"fmt"
	"log"
	"time"
	"zeus-scm-service/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func seedCatalogs(db *gorm.DB, data *PartsFile) (map[string]int32, map[string]models.PartCatalog) {
	typeMap := make(map[string]int32)
	catMap := make(map[string]models.PartCatalog)

	for i, pt := range data.PartTypes {
		id := int32(i + 1)
		desc := pt.Description
		partType := models.PartType{ID: id, PartTypeName: pt.CommodityType, Description: &desc}
		db.Where("part_type_name = ?", pt.CommodityType).Assign(partType).FirstOrCreate(&partType)
		typeMap[pt.CommodityType] = id
	}

	for _, pc := range data.PartCatalogs {
		tid, ok := typeMap[pc.CommodityType]
		if !ok {
			continue
		}
		desc := pc.Description
		cat := models.PartCatalog{
			ID:            uuid.New(),
			PartNumber:    pc.PartNumber,
			PartTypesID:   tid,
			MfgNumber:     pc.MfgNumber,
			Description:   &desc,
			PartMfgStatus: 1, // pending
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}
		db.Where("part_number = ? AND mfg_number = ?", pc.PartNumber, pc.MfgNumber).Assign(cat).FirstOrCreate(&cat)
		key := fmt.Sprintf("%s|%s", pc.PartNumber, pc.MfgNumber)
		catMap[key] = cat
	}
	if err := ensurePartsByModelTable(db); err != nil {
		log.Printf("warning: failed to ensure parts_by_model table: %v", err)
	}
	return typeMap, catMap
}
