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
			return tx.Exec("ALTER TABLE credentials ADD COLUMN IF NOT EXISTS project_id TEXT NOT NULL DEFAULT ''").Error
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.Exec("ALTER TABLE credentials DROP COLUMN IF EXISTS project_id").Error
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
	type roleRow struct {
		ID          string
		Name        string
		DisplayName string
		Description string
		Permissions string
		BuiltIn     bool
	}

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
				row := roleRow{
					ID:          api.NewID(),
					Name:        r.name,
					DisplayName: r.displayName,
					Description: r.description,
					Permissions: string(permsJSON),
					BuiltIn:     true,
				}
				if err := tx.Table("roles").
					Where("name = ?", r.name).
					FirstOrCreate(&row).Error; err != nil {
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
			return tx.Table("roles").Where("name IN ?", names).Delete(&roleRow{}).Error
		},
	}
}
