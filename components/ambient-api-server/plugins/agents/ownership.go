package agents

import (
	"context"

	"github.com/openshift-online/rh-trex-ai/pkg/errors"
)

type agentOwnershipAdapter struct {
	agentSvc AgentService
}

func (a *agentOwnershipAdapter) VerifyAgentProject(ctx context.Context, agentID, projectID string) *errors.ServiceError {
	if !validIDPattern.MatchString(agentID) {
		return errors.Validation("invalid agent id")
	}
	agent, err := a.agentSvc.Get(ctx, agentID)
	if err != nil {
		return err
	}
	if agent.ProjectId != projectID {
		return errors.Forbidden("agent does not belong to this project")
	}
	return nil
}
