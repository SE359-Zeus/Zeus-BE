package seeder

import (
	"encoding/json"
	"os"

	"github.com/google/uuid"
)

type PartTypeData struct {
	CommodityType string `json:"commodity_type"`
	Description   string `json:"description"`
}

type PartCatalogData struct {
	PartNumber    string `json:"part_number"`
	CommodityType string `json:"commodity_type"`
	MfgNumber     string `json:"mfg_number"`
	Description   string `json:"description"`
}

type PartInstallationData struct {
	PartNumber string `json:"part_number"`
	Quantity   int    `json:"quantity"`
	MfgNumber  string `json:"mfg_number"`
}

type PartsFile struct {
	PartTypes     []PartTypeData                    `json:"part_types"`
	PartCatalogs  []PartCatalogData                 `json:"part_catalogs"`
	Installations map[string][]PartInstallationData `json:"installations"`
}

func loadPartsData(path string) (*PartsFile, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var data PartsFile
	if err := json.Unmarshal(bytes, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

func stableUUID(seed string) uuid.UUID {
	return uuid.NewMD5(uuid.Nil, []byte(seed))
}
