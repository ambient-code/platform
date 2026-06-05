package cmd

import (
	"context"
	"flag"
	"fmt"

	"github.com/golang/glog"
	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"github.com/openshift-online/rh-trex-ai/pkg/config"
	"github.com/openshift-online/rh-trex-ai/pkg/db/db_session"
	"github.com/spf13/cobra"
)

func NewSeedAdminCommand() *cobra.Command {
	dbConfig := config.NewDatabaseConfig()
	var username string

	cmd := &cobra.Command{
		Use:   "seed-admin",
		Short: "Create the initial platform:admin RoleBinding",
		Long:  "Seeds the first platform:admin user. This breaks the bootstrap chicken-and-egg: RBAC endpoints are themselves gated, so the first admin cannot grant themselves access through the API.",
		Run: func(cmd *cobra.Command, args []string) {
			if err := dbConfig.ReadFiles(); err != nil {
				glog.Fatal(err)
			}

			connection := db_session.NewProdFactory(dbConfig)
			db := connection.New(context.Background())

			// Upsert user
			userID := api.NewID()
			result := db.Exec(
				`INSERT INTO users (id, username, name, created_at, updated_at)
				 VALUES (?, ?, ?, NOW(), NOW())
				 ON CONFLICT (username) WHERE deleted_at IS NULL DO NOTHING`,
				userID, username, username,
			)
			if result.Error != nil {
				glog.Fatalf("Failed to upsert user: %v", result.Error)
			}

			// Resolve actual user ID (may already exist)
			var resolvedUserID string
			if err := db.Raw(`SELECT id FROM users WHERE username = ? AND deleted_at IS NULL`, username).Scan(&resolvedUserID).Error; err != nil {
				glog.Fatalf("Failed to resolve user ID: %v", err)
			}

			// Look up platform:admin role
			var roleID string
			if err := db.Raw(`SELECT id FROM roles WHERE name = 'platform:admin' AND deleted_at IS NULL`).Scan(&roleID).Error; err != nil || roleID == "" {
				glog.Fatal("platform:admin role not found — run migrations first")
			}

			// Create global binding (idempotent)
			bindingResult := db.Exec(
				`INSERT INTO role_bindings (id, role_id, scope, user_id, created_at, updated_at)
				 SELECT ?, ?, 'global', ?, NOW(), NOW()
				 WHERE NOT EXISTS (
				   SELECT 1 FROM role_bindings
				   WHERE role_id = ? AND scope = 'global' AND user_id = ? AND deleted_at IS NULL
				 )`,
				api.NewID(), roleID, resolvedUserID, roleID, resolvedUserID,
			)
			if bindingResult.Error != nil {
				glog.Fatalf("Failed to create admin binding: %v", bindingResult.Error)
			}

			if bindingResult.RowsAffected == 0 {
				fmt.Printf("platform:admin binding already exists for user %q\n", username)
			} else {
				fmt.Printf("platform:admin binding created for user %q (id=%s)\n", username, resolvedUserID)
			}
		},
	}

	cmd.Flags().StringVar(&username, "username", "", "Username of the admin to seed (required)")
	_ = cmd.MarkFlagRequired("username")
	dbConfig.AddFlags(cmd.PersistentFlags())
	cmd.PersistentFlags().AddGoFlagSet(flag.CommandLine)
	return cmd
}
