package agentMessages

import (
	"context"

	"gorm.io/gorm/clause"

	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"github.com/openshift-online/rh-trex-ai/pkg/db"
)

type AgentMessageDao interface {
	Get(ctx context.Context, id string) (*AgentMessage, error)
	Create(ctx context.Context, agentMessage *AgentMessage) (*AgentMessage, error)
	Replace(ctx context.Context, agentMessage *AgentMessage) (*AgentMessage, error)
	Delete(ctx context.Context, id string) error
	FindByIDs(ctx context.Context, ids []string) (AgentMessageList, error)
	All(ctx context.Context) (AgentMessageList, error)
}

var _ AgentMessageDao = &sqlAgentMessageDao{}

type sqlAgentMessageDao struct {
	sessionFactory *db.SessionFactory
}

func NewAgentMessageDao(sessionFactory *db.SessionFactory) AgentMessageDao {
	return &sqlAgentMessageDao{sessionFactory: sessionFactory}
}

func (d *sqlAgentMessageDao) Get(ctx context.Context, id string) (*AgentMessage, error) {
	g2 := (*d.sessionFactory).New(ctx)
	var agentMessage AgentMessage
	if err := g2.Take(&agentMessage, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &agentMessage, nil
}

func (d *sqlAgentMessageDao) Create(ctx context.Context, agentMessage *AgentMessage) (*AgentMessage, error) {
	g2 := (*d.sessionFactory).New(ctx)
	if err := g2.Omit(clause.Associations).Create(agentMessage).Error; err != nil {
		db.MarkForRollback(ctx, err)
		return nil, err
	}
	return agentMessage, nil
}

func (d *sqlAgentMessageDao) Replace(ctx context.Context, agentMessage *AgentMessage) (*AgentMessage, error) {
	g2 := (*d.sessionFactory).New(ctx)
	if err := g2.Omit(clause.Associations).Save(agentMessage).Error; err != nil {
		db.MarkForRollback(ctx, err)
		return nil, err
	}
	return agentMessage, nil
}

func (d *sqlAgentMessageDao) Delete(ctx context.Context, id string) error {
	g2 := (*d.sessionFactory).New(ctx)
	if err := g2.Omit(clause.Associations).Delete(&AgentMessage{Meta: api.Meta{ID: id}}).Error; err != nil {
		db.MarkForRollback(ctx, err)
		return err
	}
	return nil
}

func (d *sqlAgentMessageDao) FindByIDs(ctx context.Context, ids []string) (AgentMessageList, error) {
	g2 := (*d.sessionFactory).New(ctx)
	agentMessages := AgentMessageList{}
	if err := g2.Where("id in (?)", ids).Find(&agentMessages).Error; err != nil {
		return nil, err
	}
	return agentMessages, nil
}

func (d *sqlAgentMessageDao) All(ctx context.Context) (AgentMessageList, error) {
	g2 := (*d.sessionFactory).New(ctx)
	agentMessages := AgentMessageList{}
	if err := g2.Find(&agentMessages).Error; err != nil {
		return nil, err
	}
	return agentMessages, nil
}
