package roles

import (
	"encoding/json"

	"gorm.io/gorm"

	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"github.com/openshift-online/rh-trex-ai/pkg/db"
)

func migration() *gormigrate.Migration {
	type Role struct {
		db.Model
		Name        string `gorm:"uniqueIndex;not null"`
		DisplayName *string
		Description *string
		Permissions string `gorm:"type:text"`
		BuiltIn     bool   `gorm:"default:false"`
	}

	return &gormigrate.Migration{
		ID: "202603100137",
		Migrate: func(tx *gorm.DB) error {
			if err := tx.AutoMigrate(&Role{}); err != nil {
				return err
			}
			return seedBuiltInRoles(tx)
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.Migrator().DropTable(&Role{})
		},
	}
}

func seedBuiltInRoles(tx *gorm.DB) error {
	type roleRow struct {
		ID          string
		Name        string
		DisplayName string
		Description string
		Permissions string
		BuiltIn     bool
	}

	builtInRoles := []struct {
		name        string
		displayName string
		description string
		permissions []string
	}{
		{
			name:        "platform:admin",
			displayName: "Platform Admin",
			description: "Full access to all platform resources",
			permissions: []string{"*:*"},
		},
		{
			name:        "platform:viewer",
			displayName: "Platform Viewer",
			description: "Read-only access to all platform resources",
			permissions: []string{"project:read", "project:list", "session:read", "session:list", "agent:read", "agent:list"},
		},
		{
			name:        "project:owner",
			displayName: "Project Owner",
			description: "Full access to a specific project",
			permissions: []string{"project:read", "project:update", "project:delete", "agent:*", "session:*", "session_message:*", "session_checkin:*", "project_document:*", "agent_message:*", "blackboard:*", "role_binding:*"},
		},
		{
			name:        "project:editor",
			displayName: "Project Editor",
			description: "Create and manage sessions and agents in a project",
			permissions: []string{"project:read", "agent:create", "agent:read", "agent:update", "agent:list", "agent:ignite", "session:create", "session:read", "session:update", "session:list", "session_message:*", "session_checkin:read", "session_checkin:list", "project_document:read", "project_document:create", "project_document:update", "project_document:list", "agent_message:create", "agent_message:read", "agent_message:list", "blackboard:read", "blackboard:watch"},
		},
		{
			name:        "project:viewer",
			displayName: "Project Viewer",
			description: "Read-only access to a specific project",
			permissions: []string{"project:read", "agent:read", "agent:list", "session:read", "session:list", "session_message:read", "session_message:list", "session_checkin:read", "session_checkin:list", "project_document:read", "project_document:list", "blackboard:read"},
		},
		{
			name:        "agent:operator",
			displayName: "Agent Operator",
			description: "Manage and ignite agents",
			permissions: []string{"agent:read", "agent:update", "agent:ignite", "agent:list", "session:read", "session:list"},
		},
		{
			name:        "agent:observer",
			displayName: "Agent Observer",
			description: "Read agent and session state",
			permissions: []string{"agent:read", "agent:list", "session:read", "session:list", "session_checkin:read", "session_checkin:list", "blackboard:read", "blackboard:watch"},
		},
		{
			name:        "agent:runner",
			displayName: "Agent Runner",
			description: "Runtime identity for agent pods — check in, send messages, update blackboard",
			permissions: []string{"session:read", "session_checkin:create", "session_checkin:read", "agent_message:create", "agent_message:read", "agent_message:list", "blackboard:read", "blackboard:watch"},
		},
	}

	for _, r := range builtInRoles {
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
}
