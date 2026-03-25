package projectAgents

import (
	"gorm.io/gorm"

	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/openshift-online/rh-trex-ai/pkg/db"
)

func migration() *gormigrate.Migration {
	type ProjectAgent struct {
		db.Model
		ProjectId        string
		AgentId          string
		AgentVersion     *int
		CurrentSessionId *string
	}

	return &gormigrate.Migration{
		ID: "202603200957",
		Migrate: func(tx *gorm.DB) error {
			return tx.AutoMigrate(&ProjectAgent{})
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.Migrator().DropTable(&ProjectAgent{})
		},
	}
}
