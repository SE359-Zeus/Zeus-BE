package tests

import (
	"context"
	"testing"

	"zeus-scm-service/internal/models"
	sqliteRepo "zeus-scm-service/internal/repository/sqlite"
	"zeus-scm-service/internal/service"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB() *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic(err)
	}
	return db
}

func TestVendorRouting_GetOptimalSupplier(t *testing.T) {
	db := setupTestDB()
	db.AutoMigrate(&models.Supplier{}, &models.SkuMapping{}, &models.GoodsReceipt{}, &models.GRLineItem{})
	vendorRepo := sqliteRepo.NewVendorRepository(db)
	svc := service.NewVendorService(vendorRepo)

	supplier, mapping, err := svc.GetOptimalSupplier(context.Background(), "SOC-XM100-PRO")

	assert.Error(t, err, "Should fail when no suppliers exist")
	assert.Nil(t, supplier)
	assert.Nil(t, mapping)
}

func TestVendorRouting_UpdateSupplierMetrics(t *testing.T) {
	db := setupTestDB()
	db.AutoMigrate(&models.Supplier{}, &models.GoodsReceipt{}, &models.GRLineItem{})
	vendorRepo := sqliteRepo.NewVendorRepository(db)
	svc := service.NewVendorService(vendorRepo)

	err := svc.UpdateSupplierMetrics(context.Background(), uuid.New())

	assert.NoError(t, err, "Should handle empty metrics gracefully")
}
