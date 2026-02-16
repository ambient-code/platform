package agents

import (
	"gorm.io/gorm"

	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/openshift-online/rh-trex-ai/pkg/db"
)

func migration() *gormigrate.Migration {
	type Agent struct {
		db.Model
		Name    string
		RepoUrl *string
		Prompt  *string
	}

	return &gormigrate.Migration{
		ID: "202602132212",
		Migrate: func(tx *gorm.DB) error {
			return tx.AutoMigrate(&Agent{})
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.Migrator().DropTable(&Agent{})
		},
	}
}

func constraintMigration() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "202602150002",
		Migrate: func(tx *gorm.DB) error {
			return tx.Exec(`CREATE INDEX IF NOT EXISTS idx_agents_name ON agents(name)`).Error
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.Exec(`DROP INDEX IF EXISTS idx_agents_name`).Error
		},
	}
}

func projectIdMigration() *gormigrate.Migration {
	migrateStatements := []string{
		`ALTER TABLE agents ADD COLUMN IF NOT EXISTS project_id TEXT`,
		`CREATE INDEX IF NOT EXISTS idx_agents_project_id ON agents(project_id)`,
	}
	rollbackStatements := []string{
		`DROP INDEX IF EXISTS idx_agents_project_id`,
		`ALTER TABLE agents DROP COLUMN IF EXISTS project_id`,
	}

	return &gormigrate.Migration{
		ID: "202602150030",
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
