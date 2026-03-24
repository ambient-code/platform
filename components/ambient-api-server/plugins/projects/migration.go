package projects

import (
	"gorm.io/gorm"

	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/openshift-online/rh-trex-ai/pkg/db"
)

func migration() *gormigrate.Migration {
	type Project struct {
		db.Model
		Name        string `gorm:"uniqueIndex;not null"`
		DisplayName *string
		Description *string
		Labels      *string
		Annotations *string
		Status      *string
	}

	return &gormigrate.Migration{
		ID: "202602150010",
		Migrate: func(tx *gorm.DB) error {
			return tx.AutoMigrate(&Project{})
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.Migrator().DropTable(&Project{})
		},
	}
}

func ownerUserIdMigration() *gormigrate.Migration {
	migrateStatements := []string{
		// Add owner_user_id column with a default empty string for existing rows
		`ALTER TABLE projects ADD COLUMN IF NOT EXISTS owner_user_id TEXT NOT NULL DEFAULT ''`,
		// Drop the old unique index on name only
		`DROP INDEX IF EXISTS idx_projects_name`,
		// Create composite unique index on (owner_user_id, name)
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_owner_name ON projects(owner_user_id, name)`,
	}
	rollbackStatements := []string{
		// Drop the composite unique index
		`DROP INDEX IF EXISTS idx_owner_name`,
		// Recreate the old unique index on name only
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_projects_name ON projects(name)`,
		// Drop the owner_user_id column
		`ALTER TABLE projects DROP COLUMN IF EXISTS owner_user_id`,
	}

	return &gormigrate.Migration{
		ID: "202603240001",
		Migrate: func(tx *gorm.DB) error {
			for _, stmt := range migrateStatements {
				if err := tx.Exec(stmt).Error; err != nil {
					return err
				}
			}
			return nil
		},
		Rollback: func(tx *gorm.DB) error {
			for _, stmt := range rollbackStatements {
				if err := tx.Exec(stmt).Error; err != nil {
					return err
				}
			}
			return nil
		},
	}
}
