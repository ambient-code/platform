package sessions

import (
	"context"

	"gorm.io/gorm"
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
	for i, s := range d.sessions {
		if s.ID == session.ID {
			d.sessions[i] = session
			return session, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (d *sessionDaoMock) Delete(ctx context.Context, id string) error {
	for i, s := range d.sessions {
		if s.ID == id {
			d.sessions = append(d.sessions[:i], d.sessions[i+1:]...)
			return nil
		}
	}
	return gorm.ErrRecordNotFound
}

func (d *sessionDaoMock) FindByIDs(ctx context.Context, ids []string) (SessionList, error) {
	idSet := make(map[string]bool, len(ids))
	for _, id := range ids {
		idSet[id] = true
	}
	var result SessionList
	for _, s := range d.sessions {
		if idSet[s.ID] {
			result = append(result, s)
		}
	}
	return result, nil
}

func (d *sessionDaoMock) All(ctx context.Context) (SessionList, error) {
	return d.sessions, nil
}

func (d *sessionDaoMock) AllByProjectId(ctx context.Context, projectId string) (SessionList, error) {
	var filtered SessionList
	for _, s := range d.sessions {
		if s.ProjectId != nil && *s.ProjectId == projectId {
			filtered = append(filtered, s)
		}
	}
	return filtered, nil
}
