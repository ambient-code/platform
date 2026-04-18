package inbox

import (
	"context"

	"gorm.io/gorm"

	"github.com/openshift-online/rh-trex-ai/pkg/errors"
)

var _ InboxMessageDao = &inboxMessageDaoMock{}

type inboxMessageDaoMock struct {
	inboxMessages InboxMessageList
}

func NewMockInboxMessageDao() *inboxMessageDaoMock {
	return &inboxMessageDaoMock{}
}

func (d *inboxMessageDaoMock) Get(ctx context.Context, id string) (*InboxMessage, error) {
	for _, inboxMessage := range d.inboxMessages {
		if inboxMessage.ID == id {
			return inboxMessage, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (d *inboxMessageDaoMock) Create(ctx context.Context, inboxMessage *InboxMessage) (*InboxMessage, error) {
	if inboxMessage.ID == "" {
		if err := inboxMessage.BeforeCreate(nil); err != nil {
			return nil, err
		}
	}
	d.inboxMessages = append(d.inboxMessages, inboxMessage)
	return inboxMessage, nil
}

func (d *inboxMessageDaoMock) Replace(ctx context.Context, inboxMessage *InboxMessage) (*InboxMessage, error) {
	return nil, errors.NotImplemented("InboxMessage").AsError()
}

func (d *inboxMessageDaoMock) Delete(ctx context.Context, id string) error {
	return errors.NotImplemented("InboxMessage").AsError()
}

func (d *inboxMessageDaoMock) FindByIDs(ctx context.Context, ids []string) (InboxMessageList, error) {
	return nil, errors.NotImplemented("InboxMessage").AsError()
}

func (d *inboxMessageDaoMock) All(ctx context.Context) (InboxMessageList, error) {
	return d.inboxMessages, nil
}

func (d *inboxMessageDaoMock) UnreadByAgentID(ctx context.Context, agentID string) (InboxMessageList, error) {
	var result InboxMessageList
	for _, m := range d.inboxMessages {
		if m.AgentId == agentID && (m.Read == nil || !*m.Read) {
			result = append(result, m)
		}
	}
	return result, nil
}
