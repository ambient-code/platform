package inbox

import (
	"gorm.io/gorm"

	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/openshift-online/rh-trex-ai/pkg/db"
)

func migration() *gormigrate.Migration {
	type InboxMessage struct {
		db.Model
		AgentId     string
		FromAgentId *string
		FromName    *string
		Body        string
		Read        *bool
	}

	return &gormigrate.Migration{
		ID: "202603211931",
		Migrate: func(tx *gorm.DB) error {
			return tx.AutoMigrate(&InboxMessage{})
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.Migrator().DropTable(&InboxMessage{})
		},
	}
}
