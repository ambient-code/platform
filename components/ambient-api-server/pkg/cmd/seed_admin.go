package cmd

import (
	"context"
	"flag"
	"fmt"
	"time"

	"github.com/golang/glog"
	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"github.com/openshift-online/rh-trex-ai/pkg/config"
	"github.com/openshift-online/rh-trex-ai/pkg/db/db_session"
	"github.com/spf13/cobra"
	"gorm.io/gorm/clause"
)

// seedUser is a local struct for the seed-admin command's user upsert.
type seedUser struct {
	ID        string `gorm:"primaryKey"`
	Username  string `gorm:"uniqueIndex:idx_users_username_active"`
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (seedUser) TableName() string { return "users" }

// seedRoleBinding is a local struct for the seed-admin command's binding insert.
type seedRoleBinding struct {
	ID        string  `gorm:"primaryKey"`
	RoleId    string  `gorm:"column:role_id;not null"`
	Scope     string  `gorm:"not null"`
	UserId    *string `gorm:"column:user_id"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (seedRoleBinding) TableName() string { return "role_bindings" }

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
			g := connection.New(context.Background())

			// Upsert user
			now := time.Now()
			user := seedUser{
				ID:        api.NewID(),
				Username:  username,
				Name:      username,
				CreatedAt: now,
				UpdatedAt: now,
			}
			result := g.Clauses(clause.OnConflict{DoNothing: true}).Create(&user)
			if result.Error != nil {
				glog.Fatalf("Failed to upsert user: %v", result.Error)
			}

			// Resolve actual user ID (may already exist)
			var resolvedUserID string
			if err := g.Table("users").Select("id").
				Where("username = ? AND deleted_at IS NULL", username).
				Scan(&resolvedUserID).Error; err != nil {
				glog.Fatalf("Failed to resolve user ID: %v", err)
			}

			// Look up platform:admin role
			var roleID string
			if err := g.Table("roles").Select("id").
				Where("name = ? AND deleted_at IS NULL", "platform:admin").
				Scan(&roleID).Error; err != nil || roleID == "" {
				glog.Fatal("platform:admin role not found — run migrations first")
			}

			// Create global binding (idempotent) — check existence first, then insert
			var existingCount int64
			g.Table("role_bindings").
				Where("role_id = ? AND scope = ? AND user_id = ? AND deleted_at IS NULL",
					roleID, "global", resolvedUserID).
				Count(&existingCount)

			if existingCount > 0 {
				fmt.Printf("platform:admin binding already exists for user %q\n", username)
			} else {
				binding := seedRoleBinding{
					ID:        api.NewID(),
					RoleId:    roleID,
					Scope:     "global",
					UserId:    &resolvedUserID,
					CreatedAt: now,
					UpdatedAt: now,
				}
				if err := g.Create(&binding).Error; err != nil {
					glog.Fatalf("Failed to create admin binding: %v", err)
				}
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
