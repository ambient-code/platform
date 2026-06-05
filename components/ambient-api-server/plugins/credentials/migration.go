package credentials

import (
	"encoding/json"

	"gorm.io/gorm"

	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"github.com/openshift-online/rh-trex-ai/pkg/db"
)

func migration() *gormigrate.Migration {
	type Credential struct {
		db.Model
		Name        string
		Description *string
		Provider    string
		Token       *string
		Url         *string
		Email       *string
		Labels      *string
		Annotations *string
	}

	return &gormigrate.Migration{
		ID: "202603311215",
		Migrate: func(tx *gorm.DB) error {
			return tx.AutoMigrate(&Credential{})
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.Migrator().DropTable(&Credential{})
		},
	}
}

func addProjectIDMigration() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "202604101200",
		Migrate: func(tx *gorm.DB) error {
			return tx.Exec("ALTER TABLE IF EXISTS credentials ADD COLUMN IF NOT EXISTS project_id TEXT NOT NULL DEFAULT ''").Error
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.Exec("ALTER TABLE IF EXISTS credentials DROP COLUMN IF EXISTS project_id").Error
		},
	}
}

func removeCredentialReaderRoleMigration() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "202604101201",
		Migrate: func(tx *gorm.DB) error {
			return tx.Exec("DELETE FROM roles WHERE name = 'credential:reader'").Error
		},
		Rollback: func(tx *gorm.DB) error {
			return nil
		},
	}
}

func rolesMigration() *gormigrate.Migration {
	seed := []struct {
		name        string
		displayName string
		description string
		permissions []string
	}{
		{
			name:        "credential:token-reader",
			displayName: "Credential Token Reader",
			description: "Retrieve the raw token value for a credential",
			permissions: []string{"credential:token"},
		},
		{
			name:        "credential:reader",
			displayName: "Credential Reader",
			description: "Read credential metadata (name, provider, description)",
			permissions: []string{"credential:read", "credential:list"},
		},
	}

	return &gormigrate.Migration{
		ID: "202603311216",
		Migrate: func(tx *gorm.DB) error {
			for _, r := range seed {
				permsJSON, err := json.Marshal(r.permissions)
				if err != nil {
					return err
				}
				if err := tx.Exec(
					`INSERT INTO roles (id, name, display_name, description, permissions, built_in) VALUES (?, ?, ?, ?, ?, ?) ON CONFLICT (name) DO NOTHING`,
					api.NewID(), r.name, r.displayName, r.description, string(permsJSON), true,
				).Error; err != nil {
					return err
				}
			}
			return nil
		},
		Rollback: func(tx *gorm.DB) error {
			names := make([]string, len(seed))
			for i, r := range seed {
				names[i] = r.name
			}
			return tx.Exec("DELETE FROM roles WHERE name IN ?", names).Error
		},
	}
}

func credentialOwnerRoleMigration() *gormigrate.Migration {
	seed := []struct {
		name        string
		displayName string
		description string
		permissions []string
	}{
		{
			name:        "credential:owner",
			displayName: "Credential Owner",
			description: "Full CRUD on owned credentials and bind to projects",
			permissions: []string{"credential:create", "credential:read", "credential:update", "credential:delete", "credential:list"},
		},
		{
			name:        "agent:editor",
			displayName: "Agent Editor",
			description: "Update prompt and metadata on a specific agent",
			permissions: []string{"agent:read", "agent:update"},
		},
	}

	return &gormigrate.Migration{
		ID: "202606050002",
		Migrate: func(tx *gorm.DB) error {
			for _, r := range seed {
				permsJSON, err := json.Marshal(r.permissions)
				if err != nil {
					return err
				}
				if err := tx.Exec(
					`INSERT INTO roles (id, name, display_name, description, permissions, built_in) VALUES (?, ?, ?, ?, ?, ?) ON CONFLICT (name) DO NOTHING`,
					api.NewID(), r.name, r.displayName, r.description, string(permsJSON), true,
				).Error; err != nil {
					return err
				}
			}
			return nil
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.Exec("DELETE FROM roles WHERE name IN ?", []string{"credential:owner", "agent:editor"}).Error
		},
	}
}

func credentialTokenPermMigration() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "202606050003",
		Migrate: func(tx *gorm.DB) error {
			// Add credential:fetch_token to credential:owner permissions.
			ownerPerms, _ := json.Marshal([]string{
				"credential:create", "credential:read", "credential:update",
				"credential:delete", "credential:list", "credential:fetch_token",
				"role_binding:create", "role_binding:delete",
			})
			if err := tx.Exec(
				`UPDATE roles SET permissions = ? WHERE name = 'credential:owner' AND deleted_at IS NULL`,
				string(ownerPerms),
			).Error; err != nil {
				return err
			}

			// Align credential:token-reader with the action returned by
			// pathToAction ("fetch_token") so the middleware matches.
			tokenReaderPerms, _ := json.Marshal([]string{"credential:fetch_token"})
			return tx.Exec(
				`UPDATE roles SET permissions = ? WHERE name = 'credential:token-reader' AND deleted_at IS NULL`,
				string(tokenReaderPerms),
			).Error
		},
		Rollback: func(tx *gorm.DB) error {
			ownerPerms, _ := json.Marshal([]string{
				"credential:create", "credential:read", "credential:update",
				"credential:delete", "credential:list",
			})
			if err := tx.Exec(
				`UPDATE roles SET permissions = ? WHERE name = 'credential:owner' AND deleted_at IS NULL`,
				string(ownerPerms),
			).Error; err != nil {
				return err
			}
			tokenReaderPerms, _ := json.Marshal([]string{"credential:token"})
			return tx.Exec(
				`UPDATE roles SET permissions = ? WHERE name = 'credential:token-reader' AND deleted_at IS NULL`,
				string(tokenReaderPerms),
			).Error
		},
	}
}

func credentialOwnerRoleBindingPermMigration() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "202606050004",
		Migrate: func(tx *gorm.DB) error {
			perms, _ := json.Marshal([]string{
				"credential:create", "credential:read", "credential:update",
				"credential:delete", "credential:list", "credential:fetch_token",
				"role_binding:create", "role_binding:delete",
			})
			return tx.Exec(
				`UPDATE roles SET permissions = ? WHERE name = 'credential:owner' AND deleted_at IS NULL`,
				string(perms),
			).Error
		},
		Rollback: func(tx *gorm.DB) error {
			perms, _ := json.Marshal([]string{
				"credential:create", "credential:read", "credential:update",
				"credential:delete", "credential:list", "credential:fetch_token",
			})
			return tx.Exec(
				`UPDATE roles SET permissions = ? WHERE name = 'credential:owner' AND deleted_at IS NULL`,
				string(perms),
			).Error
		},
	}
}

func dropProjectIDMigration() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "202605060003",
		Migrate: func(tx *gorm.DB) error {
			return tx.Exec(`ALTER TABLE IF EXISTS credentials DROP COLUMN IF EXISTS project_id`).Error
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.Exec(`ALTER TABLE IF EXISTS credentials ADD COLUMN IF NOT EXISTS project_id TEXT NOT NULL DEFAULT ''`).Error
		},
	}
}
