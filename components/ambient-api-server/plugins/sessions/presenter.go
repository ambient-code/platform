package sessions

import (
	"github.com/ambient/platform/components/ambient-api-server/pkg/api/openapi"
	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"github.com/openshift-online/rh-trex-ai/pkg/api/presenters"
	"github.com/openshift-online/rh-trex-ai/pkg/util"
)

func ConvertSession(session openapi.Session) *Session {
	c := &Session{
		Meta: api.Meta{
			ID: util.NilToEmptyString(session.Id),
		},
	}
	c.Name = session.Name
	c.RepoUrl = session.RepoUrl
	c.Prompt = session.Prompt
	c.AssignedUserId = session.AssignedUserId
	c.WorkflowId = session.WorkflowId
	c.Repos = session.Repos
	c.Interactive = session.Interactive
	c.Timeout = session.Timeout
	c.LlmModel = session.LlmModel
	c.LlmTemperature = session.LlmTemperature
	c.LlmMaxTokens = session.LlmMaxTokens
	c.ParentSessionId = session.ParentSessionId
	c.BotAccountName = session.BotAccountName
	c.ResourceOverrides = session.ResourceOverrides
	c.EnvironmentVariables = session.EnvironmentVariables
	c.SessionLabels = session.Labels
	c.SessionAnnotations = session.Annotations
	c.ProjectId = session.ProjectId

	if session.CreatedAt != nil {
		c.CreatedAt = *session.CreatedAt
	}
	if session.UpdatedAt != nil {
		c.UpdatedAt = *session.UpdatedAt
	}

	return c
}

func PresentSession(session *Session) openapi.Session {
	reference := presenters.PresentReference(session.ID, session)
	return openapi.Session{
		Id:                   reference.Id,
		Kind:                 reference.Kind,
		Href:                 reference.Href,
		CreatedAt:            openapi.PtrTime(session.CreatedAt),
		UpdatedAt:            openapi.PtrTime(session.UpdatedAt),
		Name:                 session.Name,
		RepoUrl:              session.RepoUrl,
		Prompt:               session.Prompt,
		CreatedByUserId:      session.CreatedByUserId,
		AssignedUserId:       session.AssignedUserId,
		WorkflowId:           session.WorkflowId,
		Repos:                session.Repos,
		Interactive:          session.Interactive,
		Timeout:              session.Timeout,
		LlmModel:             session.LlmModel,
		LlmTemperature:       session.LlmTemperature,
		LlmMaxTokens:         session.LlmMaxTokens,
		ParentSessionId:      session.ParentSessionId,
		BotAccountName:       session.BotAccountName,
		ResourceOverrides:    session.ResourceOverrides,
		EnvironmentVariables: session.EnvironmentVariables,
		Labels:               session.SessionLabels,
		Annotations:          session.SessionAnnotations,
		ProjectId:            session.ProjectId,
		Phase:                session.Phase,
		StartTime:            session.StartTime,
		CompletionTime:       session.CompletionTime,
		SdkSessionId:         session.SdkSessionId,
		SdkRestartCount:      session.SdkRestartCount,
		Conditions:           session.Conditions,
		ReconciledRepos:      session.ReconciledRepos,
		ReconciledWorkflow:   session.ReconciledWorkflow,
		KubeCrName:           session.KubeCrName,
		KubeCrUid:            session.KubeCrUid,
		KubeNamespace:        session.KubeNamespace,
	}
}
