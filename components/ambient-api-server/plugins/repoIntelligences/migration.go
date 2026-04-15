package repoIntelligences

import (
	"time"

	"gorm.io/gorm"

	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/openshift-online/rh-trex-ai/pkg/db"
)

func migration() *gormigrate.Migration {
	type RepoIntelligence struct {
		db.Model
		ProjectID           string `gorm:"not null"`
		RepoURL             string `gorm:"not null"`
		RepoBranch          string `gorm:"not null;default:'main'"`
		Summary             string `gorm:"type:text;not null"`
		Language            string `gorm:"not null"`
		Framework           *string
		BuildSystem         *string
		TestStrategy        *string  `gorm:"type:text"`
		Architecture        *string  `gorm:"type:text"`
		Conventions         *string  `gorm:"type:text"`
		Dependencies        *string  `gorm:"type:text"`
		Caveats             *string  `gorm:"type:text"`
		AnalyzedBySessionID *string
		AnalyzedByAgentID   *string
		AnalyzedAt          *time.Time
		Confidence          *float64
		Version             int `gorm:"not null;default:1"`
	}

	return &gormigrate.Migration{
		ID: "202604091200",
		Migrate: func(tx *gorm.DB) error {
			if err := tx.AutoMigrate(&RepoIntelligence{}); err != nil {
				return err
			}
			stmts := []string{
				`CREATE UNIQUE INDEX IF NOT EXISTS idx_ri_project_repo ON repo_intelligences(project_id, repo_url)`,
				`CREATE INDEX IF NOT EXISTS idx_ri_project_id ON repo_intelligences(project_id)`,
				`CREATE INDEX IF NOT EXISTS idx_ri_repo_url ON repo_intelligences(repo_url)`,
				`CREATE INDEX IF NOT EXISTS idx_ri_analyzed_by_session ON repo_intelligences(analyzed_by_session_id)`,
			}
			for _, s := range stmts {
				if err := tx.Exec(s).Error; err != nil {
					return err
				}
			}
			return nil
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.Migrator().DropTable("repo_intelligences")
		},
	}
}
