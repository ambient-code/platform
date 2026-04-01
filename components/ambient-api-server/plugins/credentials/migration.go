package credentials

import (
	"gorm.io/gorm"

	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/openshift-online/rh-trex-ai/pkg/db"
)

func migration() *gormigrate.Migration {
	type Credential struct {
		db.Model
		Name        string
		Description *string
		Provider    string
		Token       *string
		Url         *string
		Email       *string
		Labels      *string
		Annotations *string
	}

	return &gormigrate.Migration{
		ID: "202603311215",
		Migrate: func(tx *gorm.DB) error {
			return tx.AutoMigrate(&Credential{})
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.Migrator().DropTable(&Credential{})
		},
	}
}
