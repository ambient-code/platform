package workflowTasks

import (
	"gorm.io/gorm"

	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/openshift-online/rh-trex-ai/pkg/db"
)

func migration() *gormigrate.Migration {
	type WorkflowTask struct {
		db.Model
		WorkflowId string
		TaskId     string
		Position   int
	}

	return &gormigrate.Migration{
		ID: "202602132220",
		Migrate: func(tx *gorm.DB) error {
			return tx.AutoMigrate(&WorkflowTask{})
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.Migrator().DropTable(&WorkflowTask{})
		},
	}
}

func constraintMigration() *gormigrate.Migration {
	migrateStatements := []string{
		`DELETE FROM workflow_tasks WHERE workflow_id NOT IN (SELECT id FROM workflows WHERE deleted_at IS NULL)
			AND deleted_at IS NULL`,
		`DELETE FROM workflow_tasks WHERE task_id NOT IN (SELECT id FROM tasks WHERE deleted_at IS NULL)
			AND deleted_at IS NULL`,
		`DELETE FROM workflow_tasks a USING workflow_tasks b
			WHERE a.id > b.id AND a.workflow_id = b.workflow_id AND a.task_id = b.task_id
			AND a.deleted_at IS NULL AND b.deleted_at IS NULL`,
		`ALTER TABLE workflow_tasks ADD CONSTRAINT fk_workflow_tasks_workflow_id
			FOREIGN KEY (workflow_id) REFERENCES workflows(id) ON DELETE CASCADE`,
		`ALTER TABLE workflow_tasks ADD CONSTRAINT fk_workflow_tasks_task_id
			FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE`,
		`ALTER TABLE workflow_tasks ADD CONSTRAINT uq_workflow_tasks_workflow_task
			UNIQUE (workflow_id, task_id)`,
		`CREATE INDEX IF NOT EXISTS idx_workflow_tasks_workflow ON workflow_tasks(workflow_id)`,
		`CREATE INDEX IF NOT EXISTS idx_workflow_tasks_task ON workflow_tasks(task_id)`,
		`CREATE INDEX IF NOT EXISTS idx_workflow_tasks_position ON workflow_tasks(workflow_id, position)`,
	}
	rollbackStatements := []string{
		`ALTER TABLE workflow_tasks DROP CONSTRAINT IF EXISTS fk_workflow_tasks_workflow_id`,
		`ALTER TABLE workflow_tasks DROP CONSTRAINT IF EXISTS fk_workflow_tasks_task_id`,
		`ALTER TABLE workflow_tasks DROP CONSTRAINT IF EXISTS uq_workflow_tasks_workflow_task`,
		`DROP INDEX IF EXISTS idx_workflow_tasks_workflow`,
		`DROP INDEX IF EXISTS idx_workflow_tasks_task`,
		`DROP INDEX IF EXISTS idx_workflow_tasks_position`,
	}

	return &gormigrate.Migration{
		ID: "202602150008",
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
