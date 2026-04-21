package repoEvents

import (
	"gorm.io/gorm"

	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/openshift-online/rh-trex-ai/pkg/db"
)

func migration() *gormigrate.Migration {
	type RepoEvent struct {
		db.Model
		ResourceType string `gorm:"not null"`
		ResourceID   string `gorm:"not null"`
		Action       string `gorm:"not null"`
		ActorType    string `gorm:"not null"`
		ActorID      string `gorm:"not null"`
		ProjectID    string `gorm:"not null"`
		Reason       *string
		Diff         *string `gorm:"type:text"`
	}

	return &gormigrate.Migration{
		ID: "202604091202",
		Migrate: func(tx *gorm.DB) error {
			if err := tx.AutoMigrate(&RepoEvent{}); err != nil {
				return err
			}
			stmts := []string{
				`CREATE INDEX IF NOT EXISTS idx_re_resource_type ON repo_events(resource_type)`,
				`CREATE INDEX IF NOT EXISTS idx_re_resource_id ON repo_events(resource_id)`,
				`CREATE INDEX IF NOT EXISTS idx_re_project_id ON repo_events(project_id)`,
			}
			for _, s := range stmts {
				if err := tx.Exec(s).Error; err != nil {
					return err
				}
			}
			return nil
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.Migrator().DropTable("repo_events")
		},
	}
}
