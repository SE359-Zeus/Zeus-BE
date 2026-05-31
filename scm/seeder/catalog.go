package seeder

import (
	"fmt"
	"log/slog"
	"time"
	"zeus-scm-service/internal/models"

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
		// Use a stable, deterministic UUID derived from part_number+mfg_number so
		// the same part always gets the same UUID across re-seeds, ensuring the
		// exported scm-manifest.json IDs match what is stored in the database.
		stableID := stableUUID("part_catalog:" + pc.PartNumber + ":" + pc.MfgNumber)
		cat := models.PartCatalog{
			ID:            stableID,
			PartNumber:    pc.PartNumber,
			PartTypesID:   tid,
			MfgNumber:     pc.MfgNumber,
			Description:   &desc,
			ImageUrl:      partTypeImageURL(pc.CommodityType),
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

func partTypeImageURL(commodityType string) *string {
	imageMap := map[string]string{
		"A-cover assembly, LCD cover assembly":    "https://placehold.co/400x300/94a3b8/1e293b?text=LCD+Cover",
		"AC ADAPTERS":                             "https://placehold.co/400x300/fbbf24/1e293b?text=AC+Adapter",
		"C-cover with keyboard":                   "https://placehold.co/400x300/a1a1aa/1e293b?text=Keyboard+C-Cover",
		"CABLES INTERNAL":                         "https://placehold.co/400x300/6366f1/ffffff?text=Internal+Cable",
		"Cable, External, power cord, USB cable, and CRU-able internal cables": "https://placehold.co/400x300/6366f1/ffffff?text=External+Cable",
		"CAMERAS":                                 "https://placehold.co/400x300/14b8a6/ffffff?text=Webcam",
		"CARDS MISC INTERNAL":                     "https://placehold.co/400x300/8b5cf6/ffffff?text=Internal+Card",
		"CMOS BATTERIES":                          "https://placehold.co/400x300/ef4444/ffffff?text=CMOS+Battery",
		"COVERS":                                  "https://placehold.co/400x300/94a3b8/1e293b?text=Cover",
		"Consumptive Bezels":                      "https://placehold.co/400x300/94a3b8/1e293b?text=Bezel",
		"FANS":                                    "https://placehold.co/400x300/06b6d4/ffffff?text=Cooling+Fan",
		"HEAT SINKS":                              "https://placehold.co/400x300/f97316/ffffff?text=Heat+Sink",
		"KITS SCREWS AND LABELS":                  "https://placehold.co/400x300/a3a3a3/1e293b?text=Screw+Kit",
		"LCD ASSEMBLIES":                          "https://placehold.co/400x300/3b82f6/ffffff?text=LCD+Assembly",
		"LCD PANELS":                              "https://placehold.co/400x300/3b82f6/ffffff?text=LCD+Panel",
		"LCD PARTS":                               "https://placehold.co/400x300/60a5fa/ffffff?text=LCD+Parts",
		"M.2 Card":                                "https://placehold.co/400x300/10b981/ffffff?text=M.2+SSD",
		"MECHANICAL ASSEMBLIES":                   "https://placehold.co/400x300/78716c/ffffff?text=Mech+Assembly",
		"MEMORY":                                  "https://placehold.co/400x300/22c55e/ffffff?text=RAM+Memory",
		"MISC INTERNAL":                           "https://placehold.co/400x300/a3a3a3/1e293b?text=Misc+Part",
		"RECRDMEDIA":                              "https://placehold.co/400x300/be185d/ffffff?text=Recovery+Media",
		"Rechargeable Batteries , internal":       "https://placehold.co/400x300/ef4444/ffffff?text=Battery",
		"Removable tape":                          "https://placehold.co/400x300/d4d4d8/1e293b?text=Tape",
		"Reusable items (Reusable Tapes, Mylar,Sponge,thermal pad, rubber, rubber feet)": "https://placehold.co/400x300/d4d4d8/1e293b?text=Reusable+Part",
		"SPEAKERS INTERNAL":                       "https://placehold.co/400x300/e879f9/ffffff?text=Speaker",
		"SYSTEM BOARDS":                           "https://placehold.co/400x300/16a34a/ffffff?text=Motherboard",
		"Tools (special tools, drivers, etc.)":    "https://placehold.co/400x300/f59e0b/1e293b?text=Tool",
		"Wireless LAN adapters":                   "https://placehold.co/400x300/2563eb/ffffff?text=WiFi+Adapter",
	}
	if url, ok := imageMap[commodityType]; ok {
		return &url
	}
	url := "https://placehold.co/400x300/e2e8f0/1e293b?text=Component"
	return &url
}
