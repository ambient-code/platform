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
	}

	return &gormigrate.Migration{
		ID: "202602132104",
		Migrate: func(tx *gorm.DB) error {
			return tx.AutoMigrate(&User{})
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.Migrator().DropTable(&User{})
		},
	}
}

func constraintMigration() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "202602150001",
		Migrate: func(tx *gorm.DB) error {
			return tx.Exec(`CREATE INDEX IF NOT EXISTS idx_users_name ON users(name)`).Error
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.Exec(`DROP INDEX IF EXISTS idx_users_name`).Error
		},
	}
}

func groupsMigration() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "202602150034",
		Migrate: func(tx *gorm.DB) error {
			return tx.Exec(`ALTER TABLE users ADD COLUMN IF NOT EXISTS groups TEXT`).Error
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.Exec(`ALTER TABLE users DROP COLUMN IF EXISTS groups`).Error
		},
	}
}
