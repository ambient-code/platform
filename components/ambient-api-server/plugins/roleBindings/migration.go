package roleBindings

import (
	"gorm.io/gorm"

	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/openshift-online/rh-trex-ai/pkg/db"
)

func migration() *gormigrate.Migration {
	type RoleBinding struct {
		db.Model
		UserId  string `gorm:"not null;index"`
		RoleId  string `gorm:"not null;index"`
		Scope   string `gorm:"not null"`
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

func typedFKMigration() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "202505130001",
		Migrate: func(tx *gorm.DB) error {
			// Drop the old unique index that depends on scope_id before altering columns
			if err := tx.Exec(`DROP INDEX IF EXISTS idx_binding_lookup`).Error; err != nil {
				return err
			}
			// Make user_id nullable
			if err := tx.Exec(`ALTER TABLE role_bindings ALTER COLUMN user_id DROP NOT NULL`).Error; err != nil {
				return err
			}
			// Drop scope_id column (replaced by typed FKs)
			if err := tx.Exec(`ALTER TABLE role_bindings DROP COLUMN IF EXISTS scope_id`).Error; err != nil {
				return err
			}
			// Add typed FK columns
			if err := tx.Exec(`ALTER TABLE role_bindings ADD COLUMN IF NOT EXISTS project_id TEXT`).Error; err != nil {
				return err
			}
			if err := tx.Exec(`ALTER TABLE role_bindings ADD COLUMN IF NOT EXISTS agent_id TEXT`).Error; err != nil {
				return err
			}
			if err := tx.Exec(`ALTER TABLE role_bindings ADD COLUMN IF NOT EXISTS session_id TEXT`).Error; err != nil {
				return err
			}
			if err := tx.Exec(`ALTER TABLE role_bindings ADD COLUMN IF NOT EXISTS credential_id TEXT`).Error; err != nil {
				return err
			}
			// Indexes for typed FK columns
			if err := tx.Exec(`CREATE INDEX IF NOT EXISTS idx_role_bindings_project_id    ON role_bindings (project_id)`).Error; err != nil {
				return err
			}
			if err := tx.Exec(`CREATE INDEX IF NOT EXISTS idx_role_bindings_agent_id      ON role_bindings (agent_id)`).Error; err != nil {
				return err
			}
			if err := tx.Exec(`CREATE INDEX IF NOT EXISTS idx_role_bindings_session_id    ON role_bindings (session_id)`).Error; err != nil {
				return err
			}
			if err := tx.Exec(`CREATE INDEX IF NOT EXISTS idx_role_bindings_credential_id ON role_bindings (credential_id)`).Error; err != nil {
				return err
			}
			return nil
		},
		Rollback: func(tx *gorm.DB) error {
			_ = tx.Exec(`DROP INDEX IF EXISTS idx_role_bindings_credential_id`).Error
			_ = tx.Exec(`DROP INDEX IF EXISTS idx_role_bindings_session_id`).Error
			_ = tx.Exec(`DROP INDEX IF EXISTS idx_role_bindings_agent_id`).Error
			_ = tx.Exec(`DROP INDEX IF EXISTS idx_role_bindings_project_id`).Error
			_ = tx.Exec(`ALTER TABLE role_bindings DROP COLUMN IF EXISTS credential_id`).Error
			_ = tx.Exec(`ALTER TABLE role_bindings DROP COLUMN IF EXISTS session_id`).Error
			_ = tx.Exec(`ALTER TABLE role_bindings DROP COLUMN IF EXISTS agent_id`).Error
			_ = tx.Exec(`ALTER TABLE role_bindings DROP COLUMN IF EXISTS project_id`).Error
			_ = tx.Exec(`ALTER TABLE role_bindings ALTER COLUMN user_id SET NOT NULL`).Error
			return tx.Exec(`ALTER TABLE role_bindings ADD COLUMN IF NOT EXISTS scope_id TEXT`).Error
		},
	}
}
