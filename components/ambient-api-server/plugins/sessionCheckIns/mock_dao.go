package sessionCheckIns

import (
	"context"

	"gorm.io/gorm"

	"github.com/openshift-online/rh-trex-ai/pkg/errors"
)

var _ SessionCheckInDao = &sessionCheckInDaoMock{}

type sessionCheckInDaoMock struct {
	sessionCheckIns SessionCheckInList
}

func NewMockSessionCheckInDao() *sessionCheckInDaoMock {
	return &sessionCheckInDaoMock{}
}

func (d *sessionCheckInDaoMock) Get(ctx context.Context, id string) (*SessionCheckIn, error) {
	for _, sessionCheckIn := range d.sessionCheckIns {
		if sessionCheckIn.ID == id {
			return sessionCheckIn, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (d *sessionCheckInDaoMock) Create(ctx context.Context, sessionCheckIn *SessionCheckIn) (*SessionCheckIn, error) {
	d.sessionCheckIns = append(d.sessionCheckIns, sessionCheckIn)
	return sessionCheckIn, nil
}

func (d *sessionCheckInDaoMock) Replace(ctx context.Context, sessionCheckIn *SessionCheckIn) (*SessionCheckIn, error) {
	return nil, errors.NotImplemented("SessionCheckIn").AsError()
}

func (d *sessionCheckInDaoMock) Delete(ctx context.Context, id string) error {
	return errors.NotImplemented("SessionCheckIn").AsError()
}

func (d *sessionCheckInDaoMock) FindByIDs(ctx context.Context, ids []string) (SessionCheckInList, error) {
	return nil, errors.NotImplemented("SessionCheckIn").AsError()
}

func (d *sessionCheckInDaoMock) All(ctx context.Context) (SessionCheckInList, error) {
	return d.sessionCheckIns, nil
}
