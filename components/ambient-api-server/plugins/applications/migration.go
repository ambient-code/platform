package applications

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"

	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"github.com/openshift-online/rh-trex-ai/pkg/db"
)

func migration() *gormigrate.Migration {
	type Application struct {
		db.Model
		Name                  string
		SourceRepoUrl         string
		SourceTargetRevision  *string
		SourcePath            string
		DestinationAmbientUrl *string
		DestinationProject    string
		CredentialId          *string
		AutoSync              *bool
		AutoPrune             *bool
		SelfHeal              *bool
		SyncOptions           *string
		RetryLimit            *int
		SyncStatus            *string
		HealthStatus          *string
		SyncRevision          *string
		OperationPhase        *string
		OperationMessage      *string
		ResourceStatus        *string
		Conditions            *string
		Labels                *string
		Annotations           *string
		LastSyncedAt          *time.Time
	}

	return &gormigrate.Migration{
		ID: "202606102223",
		Migrate: func(tx *gorm.DB) error {
			return tx.AutoMigrate(&Application{})
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.Migrator().DropTable(&Application{})
		},
	}
}

func gitopsRolesMigration() *gormigrate.Migration {
	seed := []struct {
		name        string
		displayName string
		description string
		permissions []string
	}{
		{
			name:        "gitops:admin",
			displayName: "GitOps Admin",
			description: "Full CRUD on applications (GitOps continuous sync)",
			permissions: []string{"application:create", "application:read", "application:update", "application:delete", "application:list"},
		},
		{
			name:        "gitops:viewer",
			displayName: "GitOps Viewer",
			description: "Read-only access to applications",
			permissions: []string{"application:read", "application:list"},
		},
	}

	return &gormigrate.Migration{
		ID: "202606102224",
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
