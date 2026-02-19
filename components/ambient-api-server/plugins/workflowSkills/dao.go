package workflowSkills

import (
	"context"

	"gorm.io/gorm/clause"

	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"github.com/openshift-online/rh-trex-ai/pkg/db"
)

type WorkflowSkillDao interface {
	Get(ctx context.Context, id string) (*WorkflowSkill, error)
	Create(ctx context.Context, workflowSkill *WorkflowSkill) (*WorkflowSkill, error)
	Replace(ctx context.Context, workflowSkill *WorkflowSkill) (*WorkflowSkill, error)
	Delete(ctx context.Context, id string) error
	FindByIDs(ctx context.Context, ids []string) (WorkflowSkillList, error)
	All(ctx context.Context) (WorkflowSkillList, error)
}

var _ WorkflowSkillDao = &sqlWorkflowSkillDao{}

type sqlWorkflowSkillDao struct {
	sessionFactory *db.SessionFactory
}

func NewWorkflowSkillDao(sessionFactory *db.SessionFactory) WorkflowSkillDao {
	return &sqlWorkflowSkillDao{sessionFactory: sessionFactory}
}

func (d *sqlWorkflowSkillDao) Get(ctx context.Context, id string) (*WorkflowSkill, error) {
	g2 := (*d.sessionFactory).New(ctx)
	var workflowSkill WorkflowSkill
	if err := g2.Take(&workflowSkill, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &workflowSkill, nil
}

func (d *sqlWorkflowSkillDao) Create(ctx context.Context, workflowSkill *WorkflowSkill) (*WorkflowSkill, error) {
	g2 := (*d.sessionFactory).New(ctx)
	if err := g2.Omit(clause.Associations).Create(workflowSkill).Error; err != nil {
		db.MarkForRollback(ctx, err)
		return nil, err
	}
	return workflowSkill, nil
}

func (d *sqlWorkflowSkillDao) Replace(ctx context.Context, workflowSkill *WorkflowSkill) (*WorkflowSkill, error) {
	g2 := (*d.sessionFactory).New(ctx)
	if err := g2.Omit(clause.Associations).Save(workflowSkill).Error; err != nil {
		db.MarkForRollback(ctx, err)
		return nil, err
	}
	return workflowSkill, nil
}

func (d *sqlWorkflowSkillDao) Delete(ctx context.Context, id string) error {
	g2 := (*d.sessionFactory).New(ctx)
	if err := g2.Omit(clause.Associations).Delete(&WorkflowSkill{Meta: api.Meta{ID: id}}).Error; err != nil {
		db.MarkForRollback(ctx, err)
		return err
	}
	return nil
}

func (d *sqlWorkflowSkillDao) FindByIDs(ctx context.Context, ids []string) (WorkflowSkillList, error) {
	g2 := (*d.sessionFactory).New(ctx)
	workflowSkills := WorkflowSkillList{}
	if err := g2.Where("id in (?)", ids).Find(&workflowSkills).Error; err != nil {
		return nil, err
	}
	return workflowSkills, nil
}

func (d *sqlWorkflowSkillDao) All(ctx context.Context) (WorkflowSkillList, error) {
	g2 := (*d.sessionFactory).New(ctx)
	workflowSkills := WorkflowSkillList{}
	if err := g2.Find(&workflowSkills).Error; err != nil {
		return nil, err
	}
	return workflowSkills, nil
}
