package sqlite

import (
	"context"
	"errors"
	"time"

	rootrepo "zeus-sales-service/internal/repository"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Repository struct {
	db *gorm.DB
}

func Open(dsn string) (*Repository, error) {
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	if _, err := sqlDB.Exec("PRAGMA journal_mode=WAL;"); err != nil {
		return nil, err
	}
	repo := &Repository{db: db}
	if err := repo.EnsureSchema(context.Background()); err != nil {
		return nil, err
	}
	return repo, nil
}

func New(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (repo *Repository) Close() error {
	if repo == nil || repo.db == nil {
		return nil
	}
	sqlDB, err := repo.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

func (repo *Repository) EnsureSchema(ctx context.Context) error {
	if err := repo.db.WithContext(ctx).AutoMigrate(
		&clientRecord{},
		&salesOrderStatusRecord{},
		&salesOrderRecord{},
		&salesOrderItemRecord{},
		&inventoryReservationRecord{},
		&inventoryReservationItemRecord{},
	); err != nil {
		return err
	}
	return repo.seedStatuses(ctx)
}

func (repo *Repository) seedStatuses(ctx context.Context) error {
	statuses := defaultStatuses()
	records := make([]salesOrderStatusRecord, 0, len(statuses))
	now := time.Now().UTC()
	for _, status := range statuses {
		records = append(records, salesOrderStatusRecord{
			ID:         status.ID.String(),
			Code:       status.Code,
			Label:      status.Label,
			SortOrder:  status.SortOrder,
			IsTerminal: status.IsTerminal,
			CreatedAt:  now,
			UpdatedAt:  now,
		})
	}
	return repo.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoNothing: true,
	}).Create(&records).Error
}

func salesStatusID(code string) uuid.UUID {
	return uuid.NewSHA1(uuid.NameSpaceURL, []byte("sales-order-status:"+code))
}

func mapRecordError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return rootrepo.ErrNotFound
	}
	return err
}
