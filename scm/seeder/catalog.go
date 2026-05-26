package seeder

import (
	"fmt"
	"log/slog"
	"time"
	"zeus-scm-service/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func seedCatalogs(db *gorm.DB, data *PartsFile) (map[string]int32, map[string]models.PartCatalog) {
	typeMap := make(map[string]int32)
	catMap := make(map[string]models.PartCatalog)

	var existingCount int64
	if err := db.Model(&models.PartCatalog{}).Count(&existingCount).Error; err == nil && existingCount > 0 {
		var partTypes []models.PartType
		db.Find(&partTypes)
		for _, pt := range partTypes {
			typeMap[pt.PartTypeName] = pt.ID
		}

		var catalogs []models.PartCatalog
		db.Find(&catalogs)
		for _, c := range catalogs {
			key := fmt.Sprintf("%s|%s", c.PartNumber, c.MfgNumber)
			catMap[key] = c
		}
		if err := ensurePartsByModelTable(db); err != nil {
			slog.Warn("failed to ensure parts_by_model table",
				slog.String("service", "scm"),
				slog.String("event", "seed_warning"),
				slog.String("component", "parts_by_model"),
				slog.Any("error", err),
			)
		}
		return typeMap, catMap
	}

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
		slog.Warn("failed to ensure parts_by_model table",
			slog.String("service", "scm"),
			slog.String("event", "seed_warning"),
			slog.String("component", "parts_by_model"),
			slog.Any("error", err),
		)
	}
	return typeMap, catMap
}
