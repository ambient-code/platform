package roleBindings

import (
	"gorm.io/gorm"

	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/openshift-online/rh-trex-ai/pkg/db"
)

func migration() *gormigrate.Migration {
	type RoleBinding struct {
		db.Model
		UserId  string  `gorm:"not null;index"`
		RoleId  string  `gorm:"not null;index"`
		Scope   string  `gorm:"not null"`
		ScopeId *string
	}

	return &gormigrate.Migration{
		ID: "202603100138",
		Migrate: func(tx *gorm.DB) error {
			if err := tx.AutoMigrate(&RoleBinding{}); err != nil {
				return err
			}
			return tx.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_binding_lookup ON role_bindings (user_id, role_id, scope, COALESCE(scope_id, ''))`).Error
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.Migrator().DropTable(&RoleBinding{})
		},
	}
}
