package seeder

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"time"

	"zeus-scm-service/internal/models"

	"gorm.io/gorm"
)

type seedManifest struct {
	SchemaVersion int                    `json:"schema_version"`
	Source        string                 `json:"source"`
	GeneratedAt   time.Time              `json:"generated_at"`
	SharedActors  sharedActorsManifest   `json:"shared_actors"`
	PartTypes     []partTypeManifest     `json:"part_types"`
	PartCatalogs  []partCatalogManifest  `json:"part_catalogs"`
	ProductModels []productModelManifest `json:"product_models"`
	Products      []productManifest      `json:"products"`
	Suppliers     []supplierManifest     `json:"suppliers"`
}

type sharedActorsManifest struct {
	SCMOperatorID string `json:"scm_operator_id"`
}

type partTypeManifest struct {
	ID            int32   `json:"id"`
	CommodityType string  `json:"commodity_type"`
	Description   *string `json:"description,omitempty"`
}

type partCatalogManifest struct {
	ID            string `json:"id"`
	PartNumber    string `json:"part_number"`
	MfgNumber     string `json:"mfg_number"`
	CommodityType string `json:"commodity_type"`
	Description   string `json:"description"`
}

type productModelManifest struct {
	ModelCode   string               `json:"model_code"`
	ModelName   string               `json:"model_name"`
	Description string               `json:"description"`
	UnitPrice   float64              `json:"unit_price"`
	Bom         []productBomManifest `json:"bom"`
}

type productBomManifest struct {
	PartCatalogID string `json:"part_catalog_id"`
	PartNumber    string `json:"part_number"`
	MfgNumber     string `json:"mfg_number"`
	Quantity      int32  `json:"quantity"`
}

type productManifest struct {
	ID               string `json:"id"`
	ProductModelCode string `json:"product_model_code"`
	CustomerID       string `json:"customer_id"`
	ProductName      string `json:"product_name"`
	SerialNumber     string `json:"serial_number"`
}

type supplierManifest struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Contact     string            `json:"contact"`
	Tier        string            `json:"tier"`
	SkuMappings []skuMappingEntry `json:"sku_mappings"`
}

type skuMappingEntry struct {
	SKU          string  `json:"sku"`
	Name         string  `json:"name"`
	UnitPrice    float64 `json:"unit_price"`
	LeadTimeDays int     `json:"lead_time_days"`
	MinOrderQty  int     `json:"min_order_qty"`
}

func writeSeedManifest(db *gorm.DB, outPath string) error {
	if outPath == "" {
		return nil
	}

	manifest, err := buildSeedManifest(db)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return err
	}

	bytes, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	bytes = append(bytes, '\n')
	return os.WriteFile(outPath, bytes, 0o644)
}

func buildSeedManifest(db *gorm.DB) (*seedManifest, error) {
	manifest := &seedManifest{
		SchemaVersion: 1,
		Source:        "scm",
		GeneratedAt:   time.Now().UTC(),
		SharedActors: sharedActorsManifest{
			SCMOperatorID: stableUUID("user:scm-operator").String(),
		},
	}

	var partTypes []models.PartType
	if err := db.Order("id asc").Find(&partTypes).Error; err != nil {
		return nil, err
	}
	for _, pt := range partTypes {
		manifest.PartTypes = append(manifest.PartTypes, partTypeManifest{
			ID:            pt.ID,
			CommodityType: pt.PartTypeName,
			Description:   pt.Description,
		})
	}

	partTypeByID := make(map[int32]string, len(partTypes))
	for _, pt := range partTypes {
		partTypeByID[pt.ID] = pt.PartTypeName
	}

	var catalogs []models.PartCatalog
	if err := db.Order("part_number asc, mfg_number asc").Find(&catalogs).Error; err != nil {
		return nil, err
	}
	for _, catalog := range catalogs {
		manifest.PartCatalogs = append(manifest.PartCatalogs, partCatalogManifest{
			ID:            catalog.ID.String(),
			PartNumber:    catalog.PartNumber,
			MfgNumber:     catalog.MfgNumber,
			CommodityType: partTypeByID[catalog.PartTypesID],
			Description:   derefString(catalog.Description),
		})
	}

	partCatalogByID := make(map[string]models.PartCatalog, len(catalogs))
	for _, catalog := range catalogs {
		partCatalogByID[catalog.ID.String()] = catalog
	}

	var partModelRows []models.PartsByModel
	if err := db.Order("product_model_code asc, part_catalog_id asc").Find(&partModelRows).Error; err != nil {
		return nil, err
	}
	bomByModel := make(map[string][]productBomManifest)
	for _, row := range partModelRows {
		catalog, ok := partCatalogByID[row.PartCatalogID.String()]
		if !ok {
			continue
		}
		bomByModel[row.ProductModelCode] = append(bomByModel[row.ProductModelCode], productBomManifest{
			PartCatalogID: row.PartCatalogID.String(),
			PartNumber:    catalog.PartNumber,
			MfgNumber:     catalog.MfgNumber,
			Quantity:      row.Quantity,
		})
	}

	var productModels []models.ProductModel
	if err := db.Order("model_code asc").Find(&productModels).Error; err != nil {
		return nil, err
	}
	for _, pm := range productModels {
		manifest.ProductModels = append(manifest.ProductModels, productModelManifest{
			ModelCode:   pm.ModelCode,
			ModelName:   pm.ModelName,
			Description: derefString(pm.Description),
			UnitPrice:   pm.UnitPrice,
			Bom:         bomByModel[pm.ModelCode],
		})
	}

	var products []models.Product
	if err := db.Order("created_at asc, serial_number asc").Find(&products).Error; err != nil {
		return nil, err
	}
	for _, p := range products {
		manifest.Products = append(manifest.Products, productManifest{
			ID:               p.ID.String(),
			ProductModelCode: p.ProductModelCode,
			CustomerID:       p.CustomerID.String(),
			ProductName:      p.ProductName,
			SerialNumber:     p.SerialNumber,
		})
	}

	var suppliers []models.Supplier
	if err := db.Preload("SkuMappings", func(tx *gorm.DB) *gorm.DB {
		return tx.Order("sku asc")
	}).Order("name asc").Find(&suppliers).Error; err != nil {
		return nil, err
	}
	for _, supplier := range suppliers {
		entry := supplierManifest{
			ID:      supplier.ID.String(),
			Name:    supplier.Name,
			Contact: supplier.Contact,
			Tier:    string(supplier.Tier),
		}
		for _, mapping := range supplier.SkuMappings {
			entry.SkuMappings = append(entry.SkuMappings, skuMappingEntry{
				SKU:          mapping.SKU,
				Name:         mapping.Name,
				UnitPrice:    mapping.UnitPrice,
				LeadTimeDays: mapping.LeadTimeDays,
				MinOrderQty:  mapping.MinOrderQty,
			})
		}
		manifest.Suppliers = append(manifest.Suppliers, entry)
	}

	sort.SliceStable(manifest.ProductModels, func(i, j int) bool {
		return manifest.ProductModels[i].ModelCode < manifest.ProductModels[j].ModelCode
	})

	return manifest, nil
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
