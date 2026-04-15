package repoIntelligences

import (
	"gorm.io/gorm"

	"github.com/go-gormigrate/gormigrate/v2"
)

func migrationFixUniqueIndex() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "202604091210",
		Migrate: func(tx *gorm.DB) error {
			stmts := []string{
				// Drop the old unique index that doesn't exclude soft-deleted rows
				`DROP INDEX IF EXISTS idx_ri_project_repo`,
				// Create a partial unique index that only applies to non-deleted rows
				`CREATE UNIQUE INDEX IF NOT EXISTS idx_ri_project_repo ON repo_intelligences(project_id, repo_url) WHERE deleted_at IS NULL`,
			}
			for _, s := range stmts {
				if err := tx.Exec(s).Error; err != nil {
					return err
				}
			}
			return nil
		},
		Rollback: func(tx *gorm.DB) error {
			stmts := []string{
				`DROP INDEX IF EXISTS idx_ri_project_repo`,
				`CREATE UNIQUE INDEX IF NOT EXISTS idx_ri_project_repo ON repo_intelligences(project_id, repo_url)`,
			}
			for _, s := range stmts {
				if err := tx.Exec(s).Error; err != nil {
					return err
				}
			}
			return nil
		},
	}
}
