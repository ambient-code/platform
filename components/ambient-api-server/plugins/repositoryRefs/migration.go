package repositoryRefs

import (
	"gorm.io/gorm"

	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/openshift-online/rh-trex-ai/pkg/db"
)

func migration() *gormigrate.Migration {
	type RepositoryRef struct {
		db.Model
		Name      string
		Url       string
		Branch    *string
		Provider  *string
		Owner     *string
		RepoName  *string
		ProjectId *string
	}

	return &gormigrate.Migration{
		ID: "202602150521",
		Migrate: func(tx *gorm.DB) error {
			return tx.AutoMigrate(&RepositoryRef{})
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.Migrator().DropTable(&RepositoryRef{})
		},
	}
}
