package projectAgents

import (
	"context"

	"gorm.io/gorm/clause"

	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"github.com/openshift-online/rh-trex-ai/pkg/db"
)

type ProjectAgentDao interface {
	Get(ctx context.Context, id string) (*ProjectAgent, error)
	Create(ctx context.Context, projectAgent *ProjectAgent) (*ProjectAgent, error)
	Replace(ctx context.Context, projectAgent *ProjectAgent) (*ProjectAgent, error)
	Delete(ctx context.Context, id string) error
	FindByIDs(ctx context.Context, ids []string) (ProjectAgentList, error)
	All(ctx context.Context) (ProjectAgentList, error)
	AllByProjectID(ctx context.Context, projectID string) (ProjectAgentList, error)
}

var _ ProjectAgentDao = &sqlProjectAgentDao{}

type sqlProjectAgentDao struct {
	sessionFactory *db.SessionFactory
}

func NewProjectAgentDao(sessionFactory *db.SessionFactory) ProjectAgentDao {
	return &sqlProjectAgentDao{sessionFactory: sessionFactory}
}

func (d *sqlProjectAgentDao) Get(ctx context.Context, id string) (*ProjectAgent, error) {
	g2 := (*d.sessionFactory).New(ctx)
	var projectAgent ProjectAgent
	if err := g2.Take(&projectAgent, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &projectAgent, nil
}

func (d *sqlProjectAgentDao) Create(ctx context.Context, projectAgent *ProjectAgent) (*ProjectAgent, error) {
	g2 := (*d.sessionFactory).New(ctx)
	if err := g2.Omit(clause.Associations).Create(projectAgent).Error; err != nil {
		db.MarkForRollback(ctx, err)
		return nil, err
	}
	return projectAgent, nil
}

func (d *sqlProjectAgentDao) Replace(ctx context.Context, projectAgent *ProjectAgent) (*ProjectAgent, error) {
	g2 := (*d.sessionFactory).New(ctx)
	if err := g2.Omit(clause.Associations).Save(projectAgent).Error; err != nil {
		db.MarkForRollback(ctx, err)
		return nil, err
	}
	return projectAgent, nil
}

func (d *sqlProjectAgentDao) Delete(ctx context.Context, id string) error {
	g2 := (*d.sessionFactory).New(ctx)
	if err := g2.Omit(clause.Associations).Delete(&ProjectAgent{Meta: api.Meta{ID: id}}).Error; err != nil {
		db.MarkForRollback(ctx, err)
		return err
	}
	return nil
}

func (d *sqlProjectAgentDao) FindByIDs(ctx context.Context, ids []string) (ProjectAgentList, error) {
	g2 := (*d.sessionFactory).New(ctx)
	projectAgents := ProjectAgentList{}
	if err := g2.Where("id in (?)", ids).Find(&projectAgents).Error; err != nil {
		return nil, err
	}
	return projectAgents, nil
}

func (d *sqlProjectAgentDao) All(ctx context.Context) (ProjectAgentList, error) {
	g2 := (*d.sessionFactory).New(ctx)
	projectAgents := ProjectAgentList{}
	if err := g2.Find(&projectAgents).Error; err != nil {
		return nil, err
	}
	return projectAgents, nil
}

func (d *sqlProjectAgentDao) AllByProjectID(ctx context.Context, projectID string) (ProjectAgentList, error) {
	g2 := (*d.sessionFactory).New(ctx)
	projectAgents := ProjectAgentList{}
	if err := g2.Where("project_id = ?", projectID).Order("created_at ASC").Find(&projectAgents).Error; err != nil {
		return nil, err
	}
	return projectAgents, nil
}
