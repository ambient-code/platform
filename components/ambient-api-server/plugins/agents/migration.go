package agents

import (
	"gorm.io/gorm"

	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/openshift-online/rh-trex-ai/pkg/db"
)

func migration() *gormigrate.Migration {
	type Agent struct {
		db.Model
		ProjectId            string
		ParentAgentId        *string
		OwnerUserId          string
		Name                 string
		DisplayName          *string
		Description          *string
		Prompt               *string  `gorm:"type:text"`
		RepoUrl              *string
		WorkflowId           *string
		LlmModel             string   `gorm:"default:'sonnet'"`
		LlmTemperature       float64  `gorm:"default:0.7"`
		LlmMaxTokens         int32    `gorm:"default:4000"`
		BotAccountName       *string
		ResourceOverrides    *string
		EnvironmentVariables *string
		Labels               *string
		Annotations          *string
		CurrentSessionId     *string
	}

	return &gormigrate.Migration{
		ID: "202603100134",
		Migrate: func(tx *gorm.DB) error {
			if err := tx.AutoMigrate(&Agent{}); err != nil {
				return err
			}
			stmts := []string{
				`CREATE INDEX IF NOT EXISTS idx_agents_project_id ON agents(project_id)`,
				`CREATE INDEX IF NOT EXISTS idx_agents_parent_agent_id ON agents(parent_agent_id)`,
				`CREATE INDEX IF NOT EXISTS idx_agents_owner_user_id ON agents(owner_user_id)`,
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
