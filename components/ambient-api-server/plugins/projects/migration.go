package projects

import (
	"gorm.io/gorm"

	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/openshift-online/rh-trex-ai/pkg/db"
)

func migration() *gormigrate.Migration {
	type Project struct {
		db.Model
		Name        string `gorm:"uniqueIndex;not null"`
		DisplayName *string
		Description *string
		Labels      *string
		Annotations *string
		Status      *string
	}

	return &gormigrate.Migration{
		ID: "202602150010",
		Migrate: func(tx *gorm.DB) error {
			return tx.AutoMigrate(&Project{})
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.Migrator().DropTable(&Project{})
		},
	}
}
