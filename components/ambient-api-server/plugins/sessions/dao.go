package sessions

import (
	"context"

	"gorm.io/gorm/clause"

	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"github.com/openshift-online/rh-trex-ai/pkg/db"
)

type SessionDao interface {
	Get(ctx context.Context, id string) (*Session, error)
	Create(ctx context.Context, session *Session) (*Session, error)
	Replace(ctx context.Context, session *Session) (*Session, error)
	Delete(ctx context.Context, id string) error
	FindByIDs(ctx context.Context, ids []string) (SessionList, error)
	All(ctx context.Context) (SessionList, error)
	AllByProjectId(ctx context.Context, projectId string) (SessionList, error)
}

var _ SessionDao = &sqlSessionDao{}

type sqlSessionDao struct {
	sessionFactory *db.SessionFactory
}

func NewSessionDao(sessionFactory *db.SessionFactory) SessionDao {
	return &sqlSessionDao{sessionFactory: sessionFactory}
}

func (d *sqlSessionDao) Get(ctx context.Context, id string) (*Session, error) {
	g2 := (*d.sessionFactory).New(ctx)
	var session Session
	if err := g2.Take(&session, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &session, nil
}

func (d *sqlSessionDao) Create(ctx context.Context, session *Session) (*Session, error) {
	g2 := (*d.sessionFactory).New(ctx)
	if err := g2.Omit(clause.Associations).Create(session).Error; err != nil {
		db.MarkForRollback(ctx, err)
		return nil, err
	}
	return session, nil
}

func (d *sqlSessionDao) Replace(ctx context.Context, session *Session) (*Session, error) {
	g2 := (*d.sessionFactory).New(ctx)
	if err := g2.Omit(clause.Associations).Save(session).Error; err != nil {
		db.MarkForRollback(ctx, err)
		return nil, err
	}
	return session, nil
}

func (d *sqlSessionDao) Delete(ctx context.Context, id string) error {
	g2 := (*d.sessionFactory).New(ctx)
	if err := g2.Omit(clause.Associations).Delete(&Session{Meta: api.Meta{ID: id}}).Error; err != nil {
		db.MarkForRollback(ctx, err)
		return err
	}
	return nil
}

func (d *sqlSessionDao) FindByIDs(ctx context.Context, ids []string) (SessionList, error) {
	g2 := (*d.sessionFactory).New(ctx)
	sessions := SessionList{}
	if err := g2.Where("id in (?)", ids).Find(&sessions).Error; err != nil {
		return nil, err
	}
	return sessions, nil
}

func (d *sqlSessionDao) All(ctx context.Context) (SessionList, error) {
	g2 := (*d.sessionFactory).New(ctx)
	sessions := SessionList{}
	if err := g2.Find(&sessions).Error; err != nil {
		return nil, err
	}
	return sessions, nil
}

func (d *sqlSessionDao) AllByProjectId(ctx context.Context, projectId string) (SessionList, error) {
	g2 := (*d.sessionFactory).New(ctx)
	sessions := SessionList{}
	if err := g2.Where("project_id = ?", projectId).Find(&sessions).Error; err != nil {
		return nil, err
	}
	return sessions, nil
}
