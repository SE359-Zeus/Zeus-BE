package sqlite

import (
	"context"
	"testing"
	"zeus-mrp-service/internal/models"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	gsqlite "gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestSQLiteRepository_BOMSoftDeleteAndOverwrite(t *testing.T) {
	// 1. Open an in-memory SQLite connection
	db, err := gorm.Open(gsqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	assert.NoError(t, err)

	// 2. Create the schema manually to match migrations
	err = db.Exec(`
		CREATE TABLE bom_entries (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			parent_model_code TEXT NOT NULL,
			component_part_id TEXT NOT NULL,
			required_quantity_per_unit INTEGER NOT NULL CHECK (required_quantity_per_unit > 0),
			deleted_at DATETIME,
			UNIQUE (parent_model_code, component_part_id)
		);
	`).Error
	assert.NoError(t, err)

	repo := &sqliteMRPRepository{db: db}
	ctx := context.Background()

	modelCode := "MODEL-A"
	partID1 := uuid.New()
	partID2 := uuid.New()

	// 3. Create BOM entries
	entries := []models.BomEntry{
		{ParentModelCode: modelCode, ComponentPartID: partID1, RequiredQuantityPerUnit: 2},
		{ParentModelCode: modelCode, ComponentPartID: partID2, RequiredQuantityPerUnit: 5},
	}
	err = repo.CreateBOMEntries(ctx, entries)
	assert.NoError(t, err)

	// 4. Verify they exist
	dbBoms, err := repo.GetBOMByModelCode(ctx, modelCode)
	assert.NoError(t, err)
	assert.Len(t, dbBoms, 2)

	allBoms, err := repo.GetAllBOMs(ctx)
	assert.NoError(t, err)
	assert.Len(t, allBoms, 2)

	// 5. Soft-delete them
	err = repo.DeleteBOMEntriesByModelCode(ctx, modelCode)
	assert.NoError(t, err)

	// 6. Verify they do not appear in queries (soft-deleted)
	dbBomsAfter, err := repo.GetBOMByModelCode(ctx, modelCode)
	assert.NoError(t, err)
	assert.Len(t, dbBomsAfter, 0)

	allBomsAfter, err := repo.GetAllBOMs(ctx)
	assert.NoError(t, err)
	assert.Len(t, allBomsAfter, 0)

	// Verify that the rows are still physically in the database but have deleted_at set
	var rawCount int64
	err = db.Table("bom_entries").Count(&rawCount).Error
	assert.NoError(t, err)
	assert.Equal(t, int64(2), rawCount)

	// 7. Hard-delete and overwrite with new entries to ensure no UNIQUE constraint violation
	err = repo.HardDeleteBOMEntriesByModelCode(ctx, modelCode)
	assert.NoError(t, err)

	var rawCountAfterHard int64
	err = db.Table("bom_entries").Count(&rawCountAfterHard).Error
	assert.NoError(t, err)
	assert.Equal(t, int64(0), rawCountAfterHard)

	// Create new entries (using the same part ID to test constraint)
	newEntries := []models.BomEntry{
		{ParentModelCode: modelCode, ComponentPartID: partID1, RequiredQuantityPerUnit: 3},
		{ParentModelCode: modelCode, ComponentPartID: partID2, RequiredQuantityPerUnit: 4},
	}
	err = repo.CreateBOMEntries(ctx, newEntries)
	assert.NoError(t, err) // Should succeed without UNIQUE conflict!

	// 8. Verify new entries exist in the queries
	finalBoms, err := repo.GetBOMByModelCode(ctx, modelCode)
	assert.NoError(t, err)
	assert.Len(t, finalBoms, 2)
	assert.Equal(t, 3, finalBoms[0].RequiredQuantityPerUnit)
	assert.Equal(t, 4, finalBoms[1].RequiredQuantityPerUnit)
}
