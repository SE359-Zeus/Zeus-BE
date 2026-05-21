package sqlite

import (
	"zeus-mrp-service/internal/repository"

	"gorm.io/gorm"
)

type sqliteMRPRepository struct {
	db *gorm.DB
}

func NewSqliteMRPRepository(db *gorm.DB) repository.MRPRepository {
	return &sqliteMRPRepository{db: db}
}
