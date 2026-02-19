package agents

import (
	"context"

	"gorm.io/gorm"

	"github.com/openshift-online/rh-trex-ai/pkg/errors"
)

var _ AgentDao = &agentDaoMock{}

type agentDaoMock struct {
	agents AgentList
}

func NewMockAgentDao() *agentDaoMock {
	return &agentDaoMock{}
}

func (d *agentDaoMock) Get(ctx context.Context, id string) (*Agent, error) {
	for _, agent := range d.agents {
		if agent.ID == id {
			return agent, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (d *agentDaoMock) Create(ctx context.Context, agent *Agent) (*Agent, error) {
	d.agents = append(d.agents, agent)
	return agent, nil
}

func (d *agentDaoMock) Replace(ctx context.Context, agent *Agent) (*Agent, error) {
	return nil, errors.NotImplemented("Agent").AsError()
}

func (d *agentDaoMock) Delete(ctx context.Context, id string) error {
	return errors.NotImplemented("Agent").AsError()
}

func (d *agentDaoMock) FindByIDs(ctx context.Context, ids []string) (AgentList, error) {
	return nil, errors.NotImplemented("Agent").AsError()
}

func (d *agentDaoMock) All(ctx context.Context) (AgentList, error) {
	return d.agents, nil
}
