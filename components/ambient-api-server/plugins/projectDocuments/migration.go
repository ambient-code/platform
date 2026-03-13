package projectDocuments

import (
	"gorm.io/gorm"

	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/openshift-online/rh-trex-ai/pkg/db"
)

func migration() *gormigrate.Migration {
	type ProjectDocument struct {
		db.Model
		ProjectId string  `gorm:"not null;index"`
		Slug      string  `gorm:"not null"`
		Title     *string
		Content   *string `gorm:"type:text"`
	}

	return &gormigrate.Migration{
		ID: "202603100140",
		Migrate: func(tx *gorm.DB) error {
			if err := tx.AutoMigrate(&ProjectDocument{}); err != nil {
				return err
			}
			return tx.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_project_slug ON project_documents (project_id, slug)`).Error
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.Migrator().DropTable(&ProjectDocument{})
		},
	}
}
