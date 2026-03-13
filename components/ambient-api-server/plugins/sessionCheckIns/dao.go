package sessionCheckIns

import (
	"context"

	"gorm.io/gorm/clause"

	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"github.com/openshift-online/rh-trex-ai/pkg/db"
)

type SessionCheckInDao interface {
	Get(ctx context.Context, id string) (*SessionCheckIn, error)
	Create(ctx context.Context, sessionCheckIn *SessionCheckIn) (*SessionCheckIn, error)
	Replace(ctx context.Context, sessionCheckIn *SessionCheckIn) (*SessionCheckIn, error)
	Delete(ctx context.Context, id string) error
	FindByIDs(ctx context.Context, ids []string) (SessionCheckInList, error)
	All(ctx context.Context) (SessionCheckInList, error)
}

var _ SessionCheckInDao = &sqlSessionCheckInDao{}

type sqlSessionCheckInDao struct {
	sessionFactory *db.SessionFactory
}

func NewSessionCheckInDao(sessionFactory *db.SessionFactory) SessionCheckInDao {
	return &sqlSessionCheckInDao{sessionFactory: sessionFactory}
}

func (d *sqlSessionCheckInDao) Get(ctx context.Context, id string) (*SessionCheckIn, error) {
	g2 := (*d.sessionFactory).New(ctx)
	var sessionCheckIn SessionCheckIn
	if err := g2.Take(&sessionCheckIn, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &sessionCheckIn, nil
}

func (d *sqlSessionCheckInDao) Create(ctx context.Context, sessionCheckIn *SessionCheckIn) (*SessionCheckIn, error) {
	g2 := (*d.sessionFactory).New(ctx)
	if err := g2.Omit(clause.Associations).Create(sessionCheckIn).Error; err != nil {
		db.MarkForRollback(ctx, err)
		return nil, err
	}
	return sessionCheckIn, nil
}

func (d *sqlSessionCheckInDao) Replace(ctx context.Context, sessionCheckIn *SessionCheckIn) (*SessionCheckIn, error) {
	g2 := (*d.sessionFactory).New(ctx)
	if err := g2.Omit(clause.Associations).Save(sessionCheckIn).Error; err != nil {
		db.MarkForRollback(ctx, err)
		return nil, err
	}
	return sessionCheckIn, nil
}

func (d *sqlSessionCheckInDao) Delete(ctx context.Context, id string) error {
	g2 := (*d.sessionFactory).New(ctx)
	if err := g2.Omit(clause.Associations).Delete(&SessionCheckIn{Meta: api.Meta{ID: id}}).Error; err != nil {
		db.MarkForRollback(ctx, err)
		return err
	}
	return nil
}

func (d *sqlSessionCheckInDao) FindByIDs(ctx context.Context, ids []string) (SessionCheckInList, error) {
	g2 := (*d.sessionFactory).New(ctx)
	sessionCheckIns := SessionCheckInList{}
	if err := g2.Where("id in (?)", ids).Find(&sessionCheckIns).Error; err != nil {
		return nil, err
	}
	return sessionCheckIns, nil
}

func (d *sqlSessionCheckInDao) All(ctx context.Context) (SessionCheckInList, error) {
	g2 := (*d.sessionFactory).New(ctx)
	sessionCheckIns := SessionCheckInList{}
	if err := g2.Find(&sessionCheckIns).Error; err != nil {
		return nil, err
	}
	return sessionCheckIns, nil
}
