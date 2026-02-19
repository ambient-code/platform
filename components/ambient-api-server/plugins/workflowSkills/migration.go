package workflowSkills

import (
	"gorm.io/gorm"

	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/openshift-online/rh-trex-ai/pkg/db"
)

func migration() *gormigrate.Migration {
	type WorkflowSkill struct {
		db.Model
		WorkflowId string
		SkillId    string
		Position   int
	}

	return &gormigrate.Migration{
		ID: "202602132219",
		Migrate: func(tx *gorm.DB) error {
			return tx.AutoMigrate(&WorkflowSkill{})
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.Migrator().DropTable(&WorkflowSkill{})
		},
	}
}

func constraintMigration() *gormigrate.Migration {
	migrateStatements := []string{
		`DELETE FROM workflow_skills WHERE workflow_id NOT IN (SELECT id FROM workflows WHERE deleted_at IS NULL)
			AND deleted_at IS NULL`,
		`DELETE FROM workflow_skills WHERE skill_id NOT IN (SELECT id FROM skills WHERE deleted_at IS NULL)
			AND deleted_at IS NULL`,
		`DELETE FROM workflow_skills a USING workflow_skills b
			WHERE a.id > b.id AND a.workflow_id = b.workflow_id AND a.skill_id = b.skill_id
			AND a.deleted_at IS NULL AND b.deleted_at IS NULL`,
		`ALTER TABLE workflow_skills ADD CONSTRAINT fk_workflow_skills_workflow_id
			FOREIGN KEY (workflow_id) REFERENCES workflows(id) ON DELETE CASCADE`,
		`ALTER TABLE workflow_skills ADD CONSTRAINT fk_workflow_skills_skill_id
			FOREIGN KEY (skill_id) REFERENCES skills(id) ON DELETE CASCADE`,
		`ALTER TABLE workflow_skills ADD CONSTRAINT uq_workflow_skills_workflow_skill
			UNIQUE (workflow_id, skill_id)`,
		`CREATE INDEX IF NOT EXISTS idx_workflow_skills_workflow ON workflow_skills(workflow_id)`,
		`CREATE INDEX IF NOT EXISTS idx_workflow_skills_skill ON workflow_skills(skill_id)`,
		`CREATE INDEX IF NOT EXISTS idx_workflow_skills_position ON workflow_skills(workflow_id, position)`,
	}
	rollbackStatements := []string{
		`ALTER TABLE workflow_skills DROP CONSTRAINT IF EXISTS fk_workflow_skills_workflow_id`,
		`ALTER TABLE workflow_skills DROP CONSTRAINT IF EXISTS fk_workflow_skills_skill_id`,
		`ALTER TABLE workflow_skills DROP CONSTRAINT IF EXISTS uq_workflow_skills_workflow_skill`,
		`DROP INDEX IF EXISTS idx_workflow_skills_workflow`,
		`DROP INDEX IF EXISTS idx_workflow_skills_skill`,
		`DROP INDEX IF EXISTS idx_workflow_skills_position`,
	}

	return &gormigrate.Migration{
		ID: "202602150007",
		Migrate: func(tx *gorm.DB) error {
			for _, stmt := range migrateStatements {
				if err := tx.Exec(stmt).Error; err != nil {
					return err
				}
			}
			return nil
		},
		Rollback: func(tx *gorm.DB) error {
			for _, stmt := range rollbackStatements {
				if err := tx.Exec(stmt).Error; err != nil {
					return err
				}
			}
			return nil
		},
	}
}
