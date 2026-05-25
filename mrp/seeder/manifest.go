package seeder

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/google/uuid"
)

type manifest struct {
	SchemaVersion int               `json:"schema_version"`
	Source        string            `json:"source"`
	PartCatalogs  []manifestCatalog `json:"part_catalogs"`
	ProductModels []manifestModel   `json:"product_models"`
}

type manifestCatalog struct {
	ID            string `json:"id"`
	PartNumber    string `json:"part_number"`
	MfgNumber     string `json:"mfg_number"`
	CommodityType string `json:"commodity_type"`
}

type manifestModel struct {
	ModelCode   string            `json:"model_code"`
	ModelName   string            `json:"model_name"`
	Description string            `json:"description"`
	Bom         []manifestBomLine `json:"bom"`
}

type manifestBomLine struct {
	PartCatalogID string `json:"part_catalog_id"`
	PartNumber    string `json:"part_number"`
	MfgNumber     string `json:"mfg_number"`
	Quantity      int32  `json:"quantity"`
}

func loadManifest(path string) (*manifest, error) {
	if path == "" {
		return nil, fmt.Errorf("manifest path is required")
	}
	bytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m manifest
	if err := json.Unmarshal(bytes, &m); err != nil {
		return nil, err
	}
	if len(m.ProductModels) == 0 {
		return nil, fmt.Errorf("manifest contains no product models")
	}
	return &m, nil
}

func stableUUID(seed string) uuid.UUID {
	return uuid.NewMD5(uuid.Nil, []byte(seed))
}
