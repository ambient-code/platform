package sessions

import (
	"gorm.io/gorm"

	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/openshift-online/rh-trex-ai/pkg/db"
)

func migration() *gormigrate.Migration {
	type Session struct {
		db.Model
		Name            string
		RepoUrl         *string
		Prompt          *string
		CreatedByUserId *string
		AssignedUserId  *string
		WorkflowId      *string
	}

	return &gormigrate.Migration{
		ID: "202602132218",
		Migrate: func(tx *gorm.DB) error {
			return tx.AutoMigrate(&Session{})
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.Migrator().DropTable(&Session{})
		},
	}
}

func constraintMigration() *gormigrate.Migration {
	migrateStatements := []string{
		`CREATE INDEX IF NOT EXISTS idx_sessions_created_by ON sessions(created_by_user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_assigned_to ON sessions(assigned_user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_workflow ON sessions(workflow_id)`,
	}
	rollbackStatements := []string{
		`DROP INDEX IF EXISTS idx_sessions_created_by`,
		`DROP INDEX IF EXISTS idx_sessions_assigned_to`,
		`DROP INDEX IF EXISTS idx_sessions_workflow`,
	}

	return &gormigrate.Migration{
		ID: "202602150006",
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

func schemaExpansionMigration() *gormigrate.Migration {
	migrateStatements := []string{
		`ALTER TABLE sessions ADD COLUMN IF NOT EXISTS repos TEXT`,
		`ALTER TABLE sessions ADD COLUMN IF NOT EXISTS interactive BOOLEAN`,
		`ALTER TABLE sessions ADD COLUMN IF NOT EXISTS timeout INTEGER`,
		`ALTER TABLE sessions ADD COLUMN IF NOT EXISTS llm_model TEXT`,
		`ALTER TABLE sessions ADD COLUMN IF NOT EXISTS llm_temperature DOUBLE PRECISION`,
		`ALTER TABLE sessions ADD COLUMN IF NOT EXISTS llm_max_tokens INTEGER`,
		`ALTER TABLE sessions ADD COLUMN IF NOT EXISTS parent_session_id TEXT`,
		`ALTER TABLE sessions ADD COLUMN IF NOT EXISTS bot_account_name TEXT`,
		`ALTER TABLE sessions ADD COLUMN IF NOT EXISTS resource_overrides TEXT`,
		`ALTER TABLE sessions ADD COLUMN IF NOT EXISTS environment_variables TEXT`,
		`ALTER TABLE sessions ADD COLUMN IF NOT EXISTS labels TEXT`,
		`ALTER TABLE sessions ADD COLUMN IF NOT EXISTS annotations TEXT`,
		`ALTER TABLE sessions ADD COLUMN IF NOT EXISTS project_id TEXT`,
		`ALTER TABLE sessions ADD COLUMN IF NOT EXISTS phase TEXT`,
		`ALTER TABLE sessions ADD COLUMN IF NOT EXISTS start_time TIMESTAMPTZ`,
		`ALTER TABLE sessions ADD COLUMN IF NOT EXISTS completion_time TIMESTAMPTZ`,
		`ALTER TABLE sessions ADD COLUMN IF NOT EXISTS sdk_session_id TEXT`,
		`ALTER TABLE sessions ADD COLUMN IF NOT EXISTS sdk_restart_count INTEGER`,
		`ALTER TABLE sessions ADD COLUMN IF NOT EXISTS conditions TEXT`,
		`ALTER TABLE sessions ADD COLUMN IF NOT EXISTS reconciled_repos TEXT`,
		`ALTER TABLE sessions ADD COLUMN IF NOT EXISTS reconciled_workflow TEXT`,
		`ALTER TABLE sessions ADD COLUMN IF NOT EXISTS kube_cr_name TEXT`,
		`ALTER TABLE sessions ADD COLUMN IF NOT EXISTS kube_cr_uid TEXT`,
		`ALTER TABLE sessions ADD COLUMN IF NOT EXISTS kube_namespace TEXT`,
		`ALTER TABLE sessions ADD CONSTRAINT fk_sessions_parent_session_id
			FOREIGN KEY (parent_session_id) REFERENCES sessions(id) ON DELETE SET NULL`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_project_id ON sessions(project_id)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_phase ON sessions(phase)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_parent_session_id ON sessions(parent_session_id)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_kube_cr_name ON sessions(kube_cr_name)`,
	}
	rollbackStatements := []string{
		`ALTER TABLE sessions DROP CONSTRAINT IF EXISTS fk_sessions_parent_session_id`,
		`DROP INDEX IF EXISTS idx_sessions_project_id`,
		`DROP INDEX IF EXISTS idx_sessions_phase`,
		`DROP INDEX IF EXISTS idx_sessions_parent_session_id`,
		`DROP INDEX IF EXISTS idx_sessions_kube_cr_name`,
		`ALTER TABLE sessions DROP COLUMN IF EXISTS repos`,
		`ALTER TABLE sessions DROP COLUMN IF EXISTS interactive`,
		`ALTER TABLE sessions DROP COLUMN IF EXISTS timeout`,
		`ALTER TABLE sessions DROP COLUMN IF EXISTS llm_model`,
		`ALTER TABLE sessions DROP COLUMN IF EXISTS llm_temperature`,
		`ALTER TABLE sessions DROP COLUMN IF EXISTS llm_max_tokens`,
		`ALTER TABLE sessions DROP COLUMN IF EXISTS parent_session_id`,
		`ALTER TABLE sessions DROP COLUMN IF EXISTS bot_account_name`,
		`ALTER TABLE sessions DROP COLUMN IF EXISTS resource_overrides`,
		`ALTER TABLE sessions DROP COLUMN IF EXISTS environment_variables`,
		`ALTER TABLE sessions DROP COLUMN IF EXISTS labels`,
		`ALTER TABLE sessions DROP COLUMN IF EXISTS annotations`,
		`ALTER TABLE sessions DROP COLUMN IF EXISTS project_id`,
		`ALTER TABLE sessions DROP COLUMN IF EXISTS phase`,
		`ALTER TABLE sessions DROP COLUMN IF EXISTS start_time`,
		`ALTER TABLE sessions DROP COLUMN IF EXISTS completion_time`,
		`ALTER TABLE sessions DROP COLUMN IF EXISTS sdk_session_id`,
		`ALTER TABLE sessions DROP COLUMN IF EXISTS sdk_restart_count`,
		`ALTER TABLE sessions DROP COLUMN IF EXISTS conditions`,
		`ALTER TABLE sessions DROP COLUMN IF EXISTS reconciled_repos`,
		`ALTER TABLE sessions DROP COLUMN IF EXISTS reconciled_workflow`,
		`ALTER TABLE sessions DROP COLUMN IF EXISTS kube_cr_name`,
		`ALTER TABLE sessions DROP COLUMN IF EXISTS kube_cr_uid`,
		`ALTER TABLE sessions DROP COLUMN IF EXISTS kube_namespace`,
	}

	return &gormigrate.Migration{
		ID: "202602150040",
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
