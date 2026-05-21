package sqlite

import (
	"zeus-scm-service/internal/models"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/golang-migrate/migrate/v4/source/file"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func NewDB(path string) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	return db, nil
}

func RunMigrations(db *gorm.DB, migrationsPath string) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	driver, err := sqlite3.WithInstance(sqlDB, &sqlite3.Config{})
	if err != nil {
		return err
	}
	src, err := (&file.File{}).Open("file://" + migrationsPath)
	if err != nil {
		return err
	}
	m, err := migrate.NewWithInstance("file", src, "sqlite3", driver)
	if err != nil {
		return err
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return err
	}
	return nil
}

type EndpointRole struct {
	Method        string `gorm:"primaryKey"`
	Path          string `gorm:"primaryKey"`
	RequiredLevel string
}

func (EndpointRole) TableName() string {
	return "endpoint_roles"
}

func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&models.Supplier{},
		&models.SkuMapping{},
		&models.PurchaseOrder{},
		&models.POLineItem{},
		&models.Shipment{},
		&models.ShipmentItem{},
		&models.GoodsReceipt{},
		&models.GRLineItem{},
		&models.ComponentStock{},
		&models.Product{},
		&models.ProductModel{},
		&models.Part{},
		&models.PartCatalog{},
		&models.PartCondition{},
		&models.PartMfgStatus{},
		&models.ApiKey{},
		&models.PurchaseOrderState{},
		&models.GoodsReceiptState{},
		&models.ComponentStockState{},
		&models.ShipmentState{},
		&EndpointRole{},
	)
}
