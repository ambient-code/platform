package agents

import (
	"gorm.io/gorm"

	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/openshift-online/rh-trex-ai/pkg/db"
)

func migration() *gormigrate.Migration {
	type Agent struct {
		db.Model
		ProjectId        string
		Name             string
		Prompt           *string `gorm:"type:text"`
		CurrentSessionId *string
		Labels           *string
		Annotations      *string
	}

	return &gormigrate.Migration{
		ID: "202603211930",
		Migrate: func(tx *gorm.DB) error {
			if err := tx.AutoMigrate(&Agent{}); err != nil {
				return err
			}
			stmts := []string{
				`CREATE INDEX IF NOT EXISTS idx_agents_project_id ON agents(project_id)`,
				`CREATE INDEX IF NOT EXISTS idx_agents_current_session_id ON agents(current_session_id)`,
			}
			for _, s := range stmts {
				if err := tx.Exec(s).Error; err != nil {
					return err
				}
			}
			return nil
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.Migrator().DropTable(&Agent{})
		},
	}
}

func agentSchemaExpansionMigration() *gormigrate.Migration {
	type Agent struct {
		db.Model
		ProjectId            string
		ParentAgentId        *string  `gorm:"index"`
		OwnerUserId          *string
		Name                 string
		DisplayName          *string
		Description          *string
		Prompt               *string  `gorm:"type:text"`
		RepoUrl              *string
		WorkflowId           *string
		LlmModel             *string
		LlmTemperature       *float64
		LlmMaxTokens         *int32
		BotAccountName       *string
		ResourceOverrides    *string
		EnvironmentVariables *string
		Labels               *string
		Annotations          *string
		CurrentSessionId     *string
	}

	return &gormigrate.Migration{
		ID: "202604181000",
		Migrate: func(tx *gorm.DB) error {
			return tx.AutoMigrate(&Agent{})
		},
		Rollback: func(tx *gorm.DB) error {
			cols := []string{
				"parent_agent_id", "owner_user_id", "display_name", "description",
				"repo_url", "workflow_id", "llm_model", "llm_temperature", "llm_max_tokens",
				"bot_account_name", "resource_overrides", "environment_variables",
			}
			for _, col := range cols {
				if err := tx.Exec("ALTER TABLE agents DROP COLUMN IF EXISTS " + col).Error; err != nil {
					return err
				}
			}
			return nil
		},
	}
}
