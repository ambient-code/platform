package permissions

import (
	"gorm.io/gorm"

	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/openshift-online/rh-trex-ai/pkg/db"
)

func migration() *gormigrate.Migration {
	type Permission struct {
		db.Model
		SubjectType string
		SubjectName string
		Role        string
		ProjectId   *string
	}

	return &gormigrate.Migration{
		ID: "202602150520",
		Migrate: func(tx *gorm.DB) error {
			return tx.AutoMigrate(&Permission{})
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.Migrator().DropTable(&Permission{})
		},
	}
}
