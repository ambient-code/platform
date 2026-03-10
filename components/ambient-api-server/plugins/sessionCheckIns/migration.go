package sessionCheckIns

import (
	"gorm.io/gorm"

	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/openshift-online/rh-trex-ai/pkg/db"
)

func migration() *gormigrate.Migration {
	type SessionCheckIn struct {
		db.Model
		SessionId string  `gorm:"not null;index"`
		AgentId   string  `gorm:"not null;index"`
		Summary   *string
		Branch    *string
		Worktree  *string
		Pr        *string
		Phase     *string
		TestCount *int
		NextSteps *string
		Items     string `gorm:"type:text"`
		Questions string `gorm:"type:text"`
		Blockers  string `gorm:"type:text"`
	}

	return &gormigrate.Migration{
		ID: "202603100139",
		Migrate: func(tx *gorm.DB) error {
			return tx.AutoMigrate(&SessionCheckIn{})
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.Migrator().DropTable(&SessionCheckIn{})
		},
	}
}
