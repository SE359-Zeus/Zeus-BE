package sqlite

import (
	"context"
	"testing"
	"time"
	"zeus-mrp-service/internal/models"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	gsqlite "gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestSQLiteRepository_ShortageSoftDelete(t *testing.T) {
	// 1. Open an in-memory SQLite connection
	db, err := gorm.Open(gsqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	assert.NoError(t, err)

	// 2. Create the schema manually
	err = db.Exec(`
		CREATE TABLE shortage_logs (
			id TEXT PRIMARY KEY,
			production_order_id TEXT NOT NULL,
			part_id TEXT NOT NULL,
			shortage_qty INTEGER NOT NULL,
			resolution_status_id INTEGER NOT NULL DEFAULT 1,
			resolution_status TEXT NOT NULL DEFAULT 'planned',
			deleted_at DATETIME
		);
	`).Error
	assert.NoError(t, err)

	repo := &sqliteMRPRepository{db: db}
	ctx := context.Background()

	orderID := uuid.New()
	partID1 := uuid.New()
	partID2 := uuid.New()

	// 3. Create Shortage logs
	log1 := &models.ShortageLog{
		ID:                uuid.New(),
		ProductionOrderID: orderID,
		PartID:            partID1,
		ShortageQty:       10,
		ResolutionStatus:  models.ResolutionStatusShortage,
	}
	log2 := &models.ShortageLog{
		ID:                uuid.New(),
		ProductionOrderID: orderID,
		PartID:            partID2,
		ShortageQty:       20,
		ResolutionStatus:  models.ResolutionStatusShortage,
	}

	err = repo.CreateShortageLog(ctx, log1)
	assert.NoError(t, err)
	err = repo.CreateShortageLog(ctx, log2)
	assert.NoError(t, err)

	// 4. Verify they exist
	dbLogs, err := repo.GetShortagesByOrderID(ctx, orderID)
	assert.NoError(t, err)
	assert.Len(t, dbLogs, 2)

	// 5. Soft-delete log1
	err = repo.DeleteShortageLog(ctx, orderID, partID1)
	assert.NoError(t, err)

	// 6. Verify log1 does not appear in queries (soft-deleted) but log2 does
	dbLogsAfter, err := repo.GetShortagesByOrderID(ctx, orderID)
	assert.NoError(t, err)
	assert.Len(t, dbLogsAfter, 1)
	assert.Equal(t, partID2, dbLogsAfter[0].PartID)

	// 7. Verify aggregated shortages does not count log1
	aggResults, err := repo.GetAggregatedShortages(ctx)
	assert.NoError(t, err)
	assert.Len(t, aggResults, 1)
	assert.Equal(t, partID2, aggResults[0].PartID)

	// Verify that the row is still physically in the database but has deleted_at set
	var rawCount int64
	err = db.Table("shortage_logs").Count(&rawCount).Error
	assert.NoError(t, err)
	assert.Equal(t, int64(2), rawCount)

	var deletedAt *time.Time
	err = db.Table("shortage_logs").Select("deleted_at").Where("part_id = ?", partID1.String()).Scan(&deletedAt).Error
	assert.NoError(t, err)
	assert.NotNil(t, deletedAt)
}
