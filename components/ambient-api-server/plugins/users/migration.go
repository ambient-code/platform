package users

import (
	"gorm.io/gorm"

	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/openshift-online/rh-trex-ai/pkg/db"
)

func migration() *gormigrate.Migration {
	type User struct {
		db.Model
		Username string
		Name     string
		Email    *string
	}

	return &gormigrate.Migration{
		ID: "202602171358",
		Migrate: func(tx *gorm.DB) error {
			return tx.AutoMigrate(&User{})
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.Migrator().DropTable(&User{})
		},
	}
}

func usernameUniqueIndexMigration() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "202606050001",
		Migrate: func(tx *gorm.DB) error {
			return tx.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_users_username ON users (username) WHERE deleted_at IS NULL`).Error
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.Exec(`DROP INDEX IF EXISTS idx_users_username`).Error
		},
	}
}
