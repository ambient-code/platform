package agentMessages

import (
	"context"

	"gorm.io/gorm"

	"github.com/openshift-online/rh-trex-ai/pkg/errors"
)

var _ AgentMessageDao = &agentMessageDaoMock{}

type agentMessageDaoMock struct {
	agentMessages AgentMessageList
}

func NewMockAgentMessageDao() *agentMessageDaoMock {
	return &agentMessageDaoMock{}
}

func (d *agentMessageDaoMock) Get(ctx context.Context, id string) (*AgentMessage, error) {
	for _, agentMessage := range d.agentMessages {
		if agentMessage.ID == id {
			return agentMessage, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (d *agentMessageDaoMock) Create(ctx context.Context, agentMessage *AgentMessage) (*AgentMessage, error) {
	d.agentMessages = append(d.agentMessages, agentMessage)
	return agentMessage, nil
}

func (d *agentMessageDaoMock) Replace(ctx context.Context, agentMessage *AgentMessage) (*AgentMessage, error) {
	return nil, errors.NotImplemented("AgentMessage").AsError()
}

func (d *agentMessageDaoMock) Delete(ctx context.Context, id string) error {
	return errors.NotImplemented("AgentMessage").AsError()
}

func (d *agentMessageDaoMock) FindByIDs(ctx context.Context, ids []string) (AgentMessageList, error) {
	return nil, errors.NotImplemented("AgentMessage").AsError()
}

func (d *agentMessageDaoMock) All(ctx context.Context) (AgentMessageList, error) {
	return d.agentMessages, nil
}
