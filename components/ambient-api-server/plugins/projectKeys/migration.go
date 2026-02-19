package projectKeys

import (
	"time"

	"gorm.io/gorm"

	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/openshift-online/rh-trex-ai/pkg/db"
)

func migration() *gormigrate.Migration {
	type ProjectKey struct {
		db.Model
		Name       string
		KeyPrefix  string
		KeyHash    string
		ProjectId  *string
		ExpiresAt  *time.Time
		LastUsedAt *time.Time
	}

	return &gormigrate.Migration{
		ID: "202602150629",
		Migrate: func(tx *gorm.DB) error {
			return tx.AutoMigrate(&ProjectKey{})
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.Migrator().DropTable(&ProjectKey{})
		},
	}
}
