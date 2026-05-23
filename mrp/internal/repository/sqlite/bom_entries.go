package sqlite

import (
	"context"
	"zeus-mrp-service/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func (r *sqliteMRPRepository) CreateBOMEntries(ctx context.Context, entries []models.BomEntry) error {
	if len(entries) == 0 {
		return nil
	}

	type rec struct {
		ParentModelCode         string `gorm:"column:parent_model_code"`
		ComponentPartID         string `gorm:"column:component_part_id"`
		RequiredQuantityPerUnit int    `gorm:"column:required_quantity_per_unit"`
	}

	records := make([]rec, 0, len(entries))
	for _, e := range entries {
		records = append(records, rec{
			ParentModelCode:         e.ParentModelCode,
			ComponentPartID:         e.ComponentPartID.String(),
			RequiredQuantityPerUnit: e.RequiredQuantityPerUnit,
		})
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return tx.Table("bom_entries").Create(&records).Error
	})
}

func (r *sqliteMRPRepository) DeleteBOMEntriesByModelCode(ctx context.Context, modelCode string) error {
	if modelCode == "" {
		return nil
	}
	return r.db.WithContext(ctx).Table("bom_entries").Where("parent_model_code = ?", modelCode).Delete(nil).Error
}

func (r *sqliteMRPRepository) GetBOMByModelCode(ctx context.Context, modelCode string) ([]models.BomEntry, error) {
	if modelCode == "" {
		return []models.BomEntry{}, nil
	}

	type row struct {
		ID                      int
		ParentModelCode         string
		ComponentPartID         string
		RequiredQuantityPerUnit int
	}

	var rows []row
	err := r.db.WithContext(ctx).
		Table("bom_entries").
		Select("id, parent_model_code, component_part_id, required_quantity_per_unit").
		Where("parent_model_code = ?", modelCode).
		Order("id ASC").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}

	entries := make([]models.BomEntry, 0, len(rows))
	for _, row := range rows {
		partID, err := uuid.Parse(row.ComponentPartID)
		if err != nil {
			continue
		}

		entries = append(entries, models.BomEntry{
			ID:                      row.ID,
			ParentModelCode:         row.ParentModelCode,
			ComponentPartID:         partID,
			RequiredQuantityPerUnit: row.RequiredQuantityPerUnit,
		})
	}

	return entries, nil
}

func (r *sqliteMRPRepository) GetAllBOMs(ctx context.Context) ([]models.BomEntry, error) {
	type row struct {
		ID                      int
		ParentModelCode         string
		ComponentPartID         string
		RequiredQuantityPerUnit int
	}

	var rows []row
	err := r.db.WithContext(ctx).
		Table("bom_entries").
		Select("id, parent_model_code, component_part_id, required_quantity_per_unit").
		Order("parent_model_code ASC, id ASC").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}

	entries := make([]models.BomEntry, 0, len(rows))
	for _, rw := range rows {
		pid, err := uuid.Parse(rw.ComponentPartID)
		if err != nil {
			continue
		}
		entries = append(entries, models.BomEntry{
			ID:                      rw.ID,
			ParentModelCode:         rw.ParentModelCode,
			ComponentPartID:         pid,
			RequiredQuantityPerUnit: rw.RequiredQuantityPerUnit,
		})
	}
	return entries, nil
}

func (r *sqliteMRPRepository) GetWhereUsedByPartID(ctx context.Context, partID uuid.UUID) ([]models.BomEntry, error) {
	if partID == uuid.Nil {
		return []models.BomEntry{}, nil
	}

	type row struct {
		ID                      int
		ParentModelCode         string
		ComponentPartID         string
		RequiredQuantityPerUnit int
	}

	var rows []row
	err := r.db.WithContext(ctx).
		Table("bom_entries").
		Select("id, parent_model_code, component_part_id, required_quantity_per_unit").
		Where("component_part_id = ?", partID.String()).
		Order("id ASC").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}

	result := make([]models.BomEntry, 0, len(rows))
	for _, rw := range rows {
		pid, err := uuid.Parse(rw.ComponentPartID)
		if err != nil {
			continue
		}
		result = append(result, models.BomEntry{
			ID:                      rw.ID,
			ParentModelCode:         rw.ParentModelCode,
			ComponentPartID:         pid,
			RequiredQuantityPerUnit: rw.RequiredQuantityPerUnit,
		})
	}
	return result, nil
}
