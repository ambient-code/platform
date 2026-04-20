package inbox

import (
	"context"

	"gorm.io/gorm/clause"

	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"github.com/openshift-online/rh-trex-ai/pkg/db"
)

type InboxMessageDao interface {
	Get(ctx context.Context, id string) (*InboxMessage, error)
	Create(ctx context.Context, inboxMessage *InboxMessage) (*InboxMessage, error)
	Replace(ctx context.Context, inboxMessage *InboxMessage) (*InboxMessage, error)
	Delete(ctx context.Context, id string) error
	FindByIDs(ctx context.Context, ids []string) (InboxMessageList, error)
	All(ctx context.Context) (InboxMessageList, error)
	UnreadByAgentID(ctx context.Context, agentID string) (InboxMessageList, error)
}

var _ InboxMessageDao = &sqlInboxMessageDao{}

type sqlInboxMessageDao struct {
	sessionFactory *db.SessionFactory
}

func NewInboxMessageDao(sessionFactory *db.SessionFactory) InboxMessageDao {
	return &sqlInboxMessageDao{sessionFactory: sessionFactory}
}

func (d *sqlInboxMessageDao) Get(ctx context.Context, id string) (*InboxMessage, error) {
	g2 := (*d.sessionFactory).New(ctx)
	var inboxMessage InboxMessage
	if err := g2.Take(&inboxMessage, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &inboxMessage, nil
}

func (d *sqlInboxMessageDao) Create(ctx context.Context, inboxMessage *InboxMessage) (*InboxMessage, error) {
	g2 := (*d.sessionFactory).New(ctx)
	if err := g2.Omit(clause.Associations).Create(inboxMessage).Error; err != nil {
		db.MarkForRollback(ctx, err)
		return nil, err
	}
	return inboxMessage, nil
}

func (d *sqlInboxMessageDao) Replace(ctx context.Context, inboxMessage *InboxMessage) (*InboxMessage, error) {
	g2 := (*d.sessionFactory).New(ctx)
	if err := g2.Omit(clause.Associations).Save(inboxMessage).Error; err != nil {
		db.MarkForRollback(ctx, err)
		return nil, err
	}
	return inboxMessage, nil
}

func (d *sqlInboxMessageDao) Delete(ctx context.Context, id string) error {
	g2 := (*d.sessionFactory).New(ctx)
	if err := g2.Omit(clause.Associations).Delete(&InboxMessage{Meta: api.Meta{ID: id}}).Error; err != nil {
		db.MarkForRollback(ctx, err)
		return err
	}
	return nil
}

func (d *sqlInboxMessageDao) FindByIDs(ctx context.Context, ids []string) (InboxMessageList, error) {
	g2 := (*d.sessionFactory).New(ctx)
	inboxMessages := InboxMessageList{}
	if err := g2.Where("id in (?)", ids).Find(&inboxMessages).Error; err != nil {
		return nil, err
	}
	return inboxMessages, nil
}

func (d *sqlInboxMessageDao) All(ctx context.Context) (InboxMessageList, error) {
	g2 := (*d.sessionFactory).New(ctx)
	inboxMessages := InboxMessageList{}
	if err := g2.Find(&inboxMessages).Error; err != nil {
		return nil, err
	}
	return inboxMessages, nil
}

func (d *sqlInboxMessageDao) UnreadByAgentID(ctx context.Context, agentID string) (InboxMessageList, error) {
	g2 := (*d.sessionFactory).New(ctx)
	messages := InboxMessageList{}
	if err := g2.Where("agent_id = ? AND (read IS NULL OR read = false)", agentID).Order("created_at ASC").Limit(1000).Find(&messages).Error; err != nil {
		return nil, err
	}
	return messages, nil
}
