package seeder

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"zeus-scm-service/internal/models"
)

type procurementLineSpec struct {
	SKU            string
	Description    string
	OrderedQty     int
	ReceivedQty    int
	DefectiveQty   int
	UnitPrice      float64
	AgingSensitive bool
	ProductionDate *time.Time
}

type procurementBundle struct {
	POID         string
	VendorID     uuid.UUID
	POStatus     models.POStatus
	PaymentTerms string
	LineItems    []procurementLineSpec

	ShipmentID     string
	ShipmentStatus models.ShipmentStatus
	Carrier        string
	TrackingNo     string
	Origin         string
	ShipDate       time.Time
	ETA            time.Time

	GRID        string
	GRStatus    models.GRStatus
	ArrivalDate time.Time
	OperatorID  string
}

func seedProcurementData(db *gorm.DB, suppliers []models.Supplier, data *PartsFile) {
	var existingCount int64
	if err := db.Model(&models.PurchaseOrder{}).Count(&existingCount).Error; err == nil && existingCount > 0 {
		return
	}

	uniqueCatalogs := uniquePartCatalogs(data.PartCatalogs)
	if len(suppliers) == 0 || len(uniqueCatalogs) == 0 {
		return
	}

	mappingsBySupplier := seedSkuMappings(db, suppliers, uniqueCatalogs)
	bundles := buildProcurementBundles(suppliers, mappingsBySupplier)
	for _, bundle := range bundles {
		seedProcurementBundle(db, bundle)
	}
}

func uniquePartCatalogs(catalogs []PartCatalogData) []PartCatalogData {
	seen := make(map[string]struct{})
	unique := make([]PartCatalogData, 0, len(catalogs))
	for _, catalog := range catalogs {
		if _, ok := seen[catalog.PartNumber]; ok {
			continue
		}
		seen[catalog.PartNumber] = struct{}{}
		unique = append(unique, catalog)
	}
	return unique
}
