package sqlite

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gorm.io/gorm"
)

const (
	DirectionUp   = "up"
	DirectionDown = "down"
)

type migrationFile struct {
	version  string
	upPath   string
	downPath string
}

func ApplyMigrations(db *gorm.DB, migrationsDir, direction string) error {
	if err := ensureSchemaMigrationsTable(db); err != nil {
		return err
	}

	files, err := readMigrationFiles(migrationsDir)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return fmt.Errorf("no migration files found in %s", migrationsDir)
	}

	switch direction {
	case DirectionUp:
		return applyUp(db, files)
	case DirectionDown:
		return applyDown(db, files)
	default:
		return fmt.Errorf("unsupported migration direction %q", direction)
	}
}

func applyUp(db *gorm.DB, files []migrationFile) error {
	for _, file := range files {
		applied, err := hasMigrationBeenApplied(db, file.version)
		if err != nil {
			return err
		}
		if applied {
			continue
		}

		content, err := os.ReadFile(file.upPath)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", filepath.Base(file.upPath), err)
		}

		if err := db.WithContext(context.Background()).Transaction(func(tx *gorm.DB) error {
			if err := tx.Exec(string(content)).Error; err != nil {
				return fmt.Errorf("apply migration %s: %w", filepath.Base(file.upPath), err)
			}
			return tx.Exec("INSERT INTO schema_migrations(version) VALUES (?)", file.version).Error
		}); err != nil {
			return err
		}
	}

	return nil
}

func applyDown(db *gorm.DB, files []migrationFile) error {
	for i := len(files) - 1; i >= 0; i-- {
		file := files[i]
		applied, err := hasMigrationBeenApplied(db, file.version)
		if err != nil {
			return err
		}
		if !applied {
			continue
		}

		content, err := os.ReadFile(file.downPath)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", filepath.Base(file.downPath), err)
		}

		if err := db.WithContext(context.Background()).Transaction(func(tx *gorm.DB) error {
			if err := tx.Exec(string(content)).Error; err != nil {
				return fmt.Errorf("rollback migration %s: %w", filepath.Base(file.downPath), err)
			}
			return tx.Exec("DELETE FROM schema_migrations WHERE version = ?", file.version).Error
		}); err != nil {
			return err
		}
	}

	return nil
}

func readMigrationFiles(dir string) ([]migrationFile, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read migrations dir %s: %w", dir, err)
	}

	filesByVersion := map[string]*migrationFile{}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if filepath.Ext(name) != ".sql" {
			continue
		}

		base := strings.TrimSuffix(name, ".sql")
		switch {
		case strings.HasSuffix(base, ".up"):
			version := strings.TrimSuffix(base, ".up")
			file := filesByVersion[version]
			if file == nil {
				file = &migrationFile{version: version}
				filesByVersion[version] = file
			}
			file.upPath = filepath.Join(dir, name)
		case strings.HasSuffix(base, ".down"):
			version := strings.TrimSuffix(base, ".down")
			file := filesByVersion[version]
			if file == nil {
				file = &migrationFile{version: version}
				filesByVersion[version] = file
			}
			file.downPath = filepath.Join(dir, name)
		}
	}

	files := make([]migrationFile, 0, len(filesByVersion))
	for _, file := range filesByVersion {
		if file.upPath == "" || file.downPath == "" {
			return nil, fmt.Errorf("migration %s must have both .up.sql and .down.sql files", file.version)
		}
		files = append(files, *file)
	}

	sort.Slice(files, func(i, j int) bool { return files[i].version < files[j].version })
	return files, nil
}

func ensureSchemaMigrationsTable(db *gorm.DB) error {
	return db.Exec(`
CREATE TABLE IF NOT EXISTS schema_migrations (
    version TEXT PRIMARY KEY,
    applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
`).Error
}

func hasMigrationBeenApplied(db *gorm.DB, version string) (bool, error) {
	var count int64
	if err := db.Table("schema_migrations").Where("version = ?", version).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}
