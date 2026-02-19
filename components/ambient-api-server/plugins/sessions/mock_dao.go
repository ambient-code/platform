package sessions

import (
	"context"

	"gorm.io/gorm"

	"github.com/openshift-online/rh-trex-ai/pkg/errors"
)

var _ SessionDao = &sessionDaoMock{}

type sessionDaoMock struct {
	sessions SessionList
}

func NewMockSessionDao() *sessionDaoMock {
	return &sessionDaoMock{}
}

func (d *sessionDaoMock) Get(ctx context.Context, id string) (*Session, error) {
	for _, session := range d.sessions {
		if session.ID == id {
			return session, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (d *sessionDaoMock) Create(ctx context.Context, session *Session) (*Session, error) {
	d.sessions = append(d.sessions, session)
	return session, nil
}

func (d *sessionDaoMock) Replace(ctx context.Context, session *Session) (*Session, error) {
	return nil, errors.NotImplemented("Session").AsError()
}

func (d *sessionDaoMock) Delete(ctx context.Context, id string) error {
	return errors.NotImplemented("Session").AsError()
}

func (d *sessionDaoMock) FindByIDs(ctx context.Context, ids []string) (SessionList, error) {
	return nil, errors.NotImplemented("Session").AsError()
}

func (d *sessionDaoMock) All(ctx context.Context) (SessionList, error) {
	return d.sessions, nil
}
