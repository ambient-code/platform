package tasks

import (
	"gorm.io/gorm"

	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/openshift-online/rh-trex-ai/pkg/db"
)

func migration() *gormigrate.Migration {
	type Task struct {
		db.Model
		Name    string
		RepoUrl *string
		Prompt  *string
	}

	return &gormigrate.Migration{
		ID: "202602132216",
		Migrate: func(tx *gorm.DB) error {
			return tx.AutoMigrate(&Task{})
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.Migrator().DropTable(&Task{})
		},
	}
}

func constraintMigration() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "202602150004",
		Migrate: func(tx *gorm.DB) error {
			return tx.Exec(`CREATE INDEX IF NOT EXISTS idx_tasks_name ON tasks(name)`).Error
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.Exec(`DROP INDEX IF EXISTS idx_tasks_name`).Error
		},
	}
}

func projectIdMigration() *gormigrate.Migration {
	migrateStatements := []string{
		`ALTER TABLE tasks ADD COLUMN IF NOT EXISTS project_id TEXT`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_project_id ON tasks(project_id)`,
	}
	rollbackStatements := []string{
		`DROP INDEX IF EXISTS idx_tasks_project_id`,
		`ALTER TABLE tasks DROP COLUMN IF EXISTS project_id`,
	}

	return &gormigrate.Migration{
		ID: "202602150032",
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
