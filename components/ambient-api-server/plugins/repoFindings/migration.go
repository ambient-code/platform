package repoFindings

import (
	"gorm.io/gorm"

	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/openshift-online/rh-trex-ai/pkg/db"
)

func migration() *gormigrate.Migration {
	type RepoFinding struct {
		db.Model
		IntelligenceID string  `gorm:"not null"`
		FilePath       string  `gorm:"not null"`
		Category       string  `gorm:"not null"`
		Status         string  `gorm:"not null;default:'active'"`
		Title          string  `gorm:"not null"`
		Body           string  `gorm:"type:text;not null"`
		Severity       *string
		Confidence     *float64
		SourceType     string  `gorm:"not null"`
		SourceRef      *string
		SessionID      *string
		AgentID        *string
		ResolvedBy     *string
		ResolvedReason *string
	}

	return &gormigrate.Migration{
		ID: "202604091201",
		Migrate: func(tx *gorm.DB) error {
			if err := tx.AutoMigrate(&RepoFinding{}); err != nil {
				return err
			}
			stmts := []string{
				`CREATE INDEX IF NOT EXISTS idx_rf_intelligence_id ON repo_findings(intelligence_id)`,
				`CREATE INDEX IF NOT EXISTS idx_rf_file_path ON repo_findings(file_path)`,
				`CREATE INDEX IF NOT EXISTS idx_rf_category ON repo_findings(category)`,
				`CREATE INDEX IF NOT EXISTS idx_rf_status ON repo_findings(status)`,
				`CREATE INDEX IF NOT EXISTS idx_rf_session_id ON repo_findings(session_id)`,
			}
			for _, s := range stmts {
				if err := tx.Exec(s).Error; err != nil {
					return err
				}
			}
			return nil
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.Migrator().DropTable("repo_findings")
		},
	}
}
