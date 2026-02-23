package projectSettings

import (
	"gorm.io/gorm"

	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/openshift-online/rh-trex-ai/pkg/db"
)

func migration() *gormigrate.Migration {
	type ProjectSettings struct {
		db.Model
		ProjectId     string `gorm:"uniqueIndex;not null"`
		GroupAccess   *string
		RunnerSecrets *string
		Repositories  *string
	}

	return &gormigrate.Migration{
		ID: "202602150020",
		Migrate: func(tx *gorm.DB) error {
			return tx.AutoMigrate(&ProjectSettings{})
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.Migrator().DropTable(&ProjectSettings{})
		},
	}
}

func constraintMigration() *gormigrate.Migration {
	migrateStatements := []string{
		`DELETE FROM project_settings WHERE project_id NOT IN (SELECT id FROM projects WHERE deleted_at IS NULL)`,
		`ALTER TABLE project_settings ADD CONSTRAINT fk_project_settings_project_id
			FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE`,
		`CREATE INDEX IF NOT EXISTS idx_project_settings_project_id ON project_settings(project_id)`,
	}
	rollbackStatements := []string{
		`ALTER TABLE project_settings DROP CONSTRAINT IF EXISTS fk_project_settings_project_id`,
		`DROP INDEX IF EXISTS idx_project_settings_project_id`,
	}
	return &gormigrate.Migration{
		ID: "202602150021",
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
