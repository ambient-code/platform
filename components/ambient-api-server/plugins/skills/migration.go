package skills

import (
	"gorm.io/gorm"

	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/openshift-online/rh-trex-ai/pkg/db"
)

func migration() *gormigrate.Migration {
	type Skill struct {
		db.Model
		Name    string
		RepoUrl *string
		Prompt  *string
	}

	return &gormigrate.Migration{
		ID: "202602132215",
		Migrate: func(tx *gorm.DB) error {
			return tx.AutoMigrate(&Skill{})
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.Migrator().DropTable(&Skill{})
		},
	}
}

func constraintMigration() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "202602150003",
		Migrate: func(tx *gorm.DB) error {
			return tx.Exec(`CREATE INDEX IF NOT EXISTS idx_skills_name ON skills(name)`).Error
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.Exec(`DROP INDEX IF EXISTS idx_skills_name`).Error
		},
	}
}

func projectIdMigration() *gormigrate.Migration {
	migrateStatements := []string{
		`ALTER TABLE skills ADD COLUMN IF NOT EXISTS project_id TEXT`,
		`CREATE INDEX IF NOT EXISTS idx_skills_project_id ON skills(project_id)`,
	}
	rollbackStatements := []string{
		`DROP INDEX IF EXISTS idx_skills_project_id`,
		`ALTER TABLE skills DROP COLUMN IF EXISTS project_id`,
	}

	return &gormigrate.Migration{
		ID: "202602150031",
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
