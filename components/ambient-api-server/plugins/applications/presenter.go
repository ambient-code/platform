package applications

import (
	"github.com/ambient-code/platform/components/ambient-api-server/pkg/api/openapi"
	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"github.com/openshift-online/rh-trex-ai/pkg/api/presenters"
	"github.com/openshift-online/rh-trex-ai/pkg/util"
)

func ConvertApplication(application openapi.Application) *Application {
	c := &Application{
		Meta: api.Meta{
			ID: util.NilToEmptyString(application.Id),
		},
	}
	c.Name = application.Name
	c.SourceRepoUrl = application.SourceRepoUrl
	c.SourceTargetRevision = application.SourceTargetRevision
	c.SourcePath = application.SourcePath
	c.DestinationAmbientUrl = application.DestinationAmbientUrl
	c.DestinationProject = application.DestinationProject
	c.CredentialId = application.CredentialId
	c.AutoSync = application.AutoSync
	c.AutoPrune = application.AutoPrune
	c.SelfHeal = application.SelfHeal
	c.SyncOptions = application.SyncOptions
	if application.RetryLimit != nil {
		c.RetryLimit = openapi.PtrInt(int(*application.RetryLimit))
	}
	c.SyncStatus = application.SyncStatus
	c.HealthStatus = application.HealthStatus
	c.SyncRevision = application.SyncRevision
	c.OperationPhase = application.OperationPhase
	c.OperationMessage = application.OperationMessage
	c.ResourceStatus = application.ResourceStatus
	c.Conditions = application.Conditions
	c.Labels = application.Labels
	c.Annotations = application.Annotations
	c.LastSyncedAt = application.LastSyncedAt

	if application.CreatedAt != nil {
		c.CreatedAt = *application.CreatedAt
		c.UpdatedAt = *application.UpdatedAt
	}

	return c
}

func PresentApplication(application *Application) openapi.Application {
	reference := presenters.PresentReference(application.ID, application)
	return openapi.Application{
		Id:                    reference.Id,
		Kind:                  reference.Kind,
		Href:                  reference.Href,
		CreatedAt:             openapi.PtrTime(application.CreatedAt),
		UpdatedAt:             openapi.PtrTime(application.UpdatedAt),
		Name:                  application.Name,
		SourceRepoUrl:         application.SourceRepoUrl,
		SourceTargetRevision:  application.SourceTargetRevision,
		SourcePath:            application.SourcePath,
		DestinationAmbientUrl: application.DestinationAmbientUrl,
		DestinationProject:    application.DestinationProject,
		CredentialId:          application.CredentialId,
		AutoSync:              application.AutoSync,
		AutoPrune:             application.AutoPrune,
		SelfHeal:              application.SelfHeal,
		SyncOptions:           application.SyncOptions,
		RetryLimit: func() *int32 {
			if application.RetryLimit != nil {
				return openapi.PtrInt32(int32(*application.RetryLimit))
			}
			return nil
		}(),
		SyncStatus:       application.SyncStatus,
		HealthStatus:     application.HealthStatus,
		SyncRevision:     application.SyncRevision,
		OperationPhase:   application.OperationPhase,
		OperationMessage: application.OperationMessage,
		ResourceStatus:   application.ResourceStatus,
		Conditions:       application.Conditions,
		Labels:           application.Labels,
		Annotations:      application.Annotations,
		LastSyncedAt:     application.LastSyncedAt,
	}
}
