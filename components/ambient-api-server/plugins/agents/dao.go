package agents

import (
	"context"

	"gorm.io/gorm/clause"

	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"github.com/openshift-online/rh-trex-ai/pkg/db"
)

type AgentDao interface {
	Get(ctx context.Context, id string) (*Agent, error)
	Create(ctx context.Context, agent *Agent) (*Agent, error)
	Replace(ctx context.Context, agent *Agent) (*Agent, error)
	Delete(ctx context.Context, id string) error
	FindByIDs(ctx context.Context, ids []string) (AgentList, error)
	All(ctx context.Context) (AgentList, error)
}

var _ AgentDao = &sqlAgentDao{}

type sqlAgentDao struct {
	sessionFactory *db.SessionFactory
}

func NewAgentDao(sessionFactory *db.SessionFactory) AgentDao {
	return &sqlAgentDao{sessionFactory: sessionFactory}
}

func (d *sqlAgentDao) Get(ctx context.Context, id string) (*Agent, error) {
	g2 := (*d.sessionFactory).New(ctx)
	var agent Agent
	if err := g2.Take(&agent, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &agent, nil
}

func (d *sqlAgentDao) Create(ctx context.Context, agent *Agent) (*Agent, error) {
	g2 := (*d.sessionFactory).New(ctx)
	if err := g2.Omit(clause.Associations).Create(agent).Error; err != nil {
		db.MarkForRollback(ctx, err)
		return nil, err
	}
	return agent, nil
}

func (d *sqlAgentDao) Replace(ctx context.Context, agent *Agent) (*Agent, error) {
	g2 := (*d.sessionFactory).New(ctx)
	if err := g2.Omit(clause.Associations).Save(agent).Error; err != nil {
		db.MarkForRollback(ctx, err)
		return nil, err
	}
	return agent, nil
}

func (d *sqlAgentDao) Delete(ctx context.Context, id string) error {
	g2 := (*d.sessionFactory).New(ctx)
	if err := g2.Omit(clause.Associations).Delete(&Agent{Meta: api.Meta{ID: id}}).Error; err != nil {
		db.MarkForRollback(ctx, err)
		return err
	}
	return nil
}

func (d *sqlAgentDao) FindByIDs(ctx context.Context, ids []string) (AgentList, error) {
	g2 := (*d.sessionFactory).New(ctx)
	agents := AgentList{}
	if err := g2.Where("id in (?)", ids).Find(&agents).Error; err != nil {
		return nil, err
	}
	return agents, nil
}

func (d *sqlAgentDao) All(ctx context.Context) (AgentList, error) {
	g2 := (*d.sessionFactory).New(ctx)
	agents := AgentList{}
	if err := g2.Find(&agents).Error; err != nil {
		return nil, err
	}
	return agents, nil
}
