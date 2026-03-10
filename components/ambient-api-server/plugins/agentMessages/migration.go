package agentMessages

import (
	"gorm.io/gorm"

	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/openshift-online/rh-trex-ai/pkg/db"
)

func migration() *gormigrate.Migration {
	type AgentMessage struct {
		db.Model
		RecipientAgentId string  `gorm:"not null;index"`
		SenderAgentId    *string
		SenderUserId     *string
		SenderName       *string
		Body             *string `gorm:"type:text"`
		Read             *bool   `gorm:"default:false"`
	}

	return &gormigrate.Migration{
		ID: "202603100141",
		Migrate: func(tx *gorm.DB) error {
			return tx.AutoMigrate(&AgentMessage{})
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.Migrator().DropTable(&AgentMessage{})
		},
	}
}
