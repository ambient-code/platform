package workflows

import (
	"gorm.io/gorm"

	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/openshift-online/rh-trex-ai/pkg/db"
)

func migration() *gormigrate.Migration {
	type Workflow struct {
		db.Model
		Name    string
		RepoUrl *string
		Prompt  *string
		AgentId *string
	}

	return &gormigrate.Migration{
		ID: "202602132217",
		Migrate: func(tx *gorm.DB) error {
			return tx.AutoMigrate(&Workflow{})
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.Migrator().DropTable(&Workflow{})
		},
	}
}

func constraintMigration() *gormigrate.Migration {
	migrateStatements := []string{
		`UPDATE workflows SET agent_id = NULL WHERE agent_id IS NOT NULL
			AND agent_id NOT IN (SELECT id FROM agents WHERE deleted_at IS NULL)
			AND deleted_at IS NULL`,
		`ALTER TABLE workflows ADD CONSTRAINT fk_workflows_agent_id
			FOREIGN KEY (agent_id) REFERENCES agents(id) ON DELETE SET NULL`,
		`CREATE INDEX IF NOT EXISTS idx_workflows_name ON workflows(name)`,
		`CREATE INDEX IF NOT EXISTS idx_workflows_agent_id ON workflows(agent_id)`,
	}
	rollbackStatements := []string{
		`ALTER TABLE workflows DROP CONSTRAINT IF EXISTS fk_workflows_agent_id`,
		`DROP INDEX IF EXISTS idx_workflows_name`,
		`DROP INDEX IF EXISTS idx_workflows_agent_id`,
	}

	return &gormigrate.Migration{
		ID: "202602150005",
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

func projectIdMigration() *gormigrate.Migration {
	migrateStatements := []string{
		`ALTER TABLE workflows ADD COLUMN IF NOT EXISTS project_id TEXT`,
		`ALTER TABLE workflows ADD COLUMN IF NOT EXISTS branch TEXT`,
		`ALTER TABLE workflows ADD COLUMN IF NOT EXISTS path TEXT`,
		`CREATE INDEX IF NOT EXISTS idx_workflows_project_id ON workflows(project_id)`,
	}
	rollbackStatements := []string{
		`DROP INDEX IF EXISTS idx_workflows_project_id`,
		`ALTER TABLE workflows DROP COLUMN IF EXISTS project_id`,
		`ALTER TABLE workflows DROP COLUMN IF EXISTS branch`,
		`ALTER TABLE workflows DROP COLUMN IF EXISTS path`,
	}

	return &gormigrate.Migration{
		ID: "202602150033",
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
