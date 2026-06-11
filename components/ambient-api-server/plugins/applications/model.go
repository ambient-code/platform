package applications

import (
	"time"

	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"gorm.io/gorm"
)

type Application struct {
	api.Meta
	Name                  string     `json:"name"`
	SourceRepoUrl         string     `json:"source_repo_url"`
	SourceTargetRevision  *string    `json:"source_target_revision"`
	SourcePath            string     `json:"source_path"`
	DestinationAmbientUrl *string    `json:"destination_ambient_url"`
	DestinationProject    string     `json:"destination_project"`
	CredentialId          *string    `json:"credential_id"`
	AutoSync              *bool      `json:"auto_sync"`
	AutoPrune             *bool      `json:"auto_prune"`
	SelfHeal              *bool      `json:"self_heal"`
	SyncOptions           *string    `json:"sync_options"`
	RetryLimit            *int       `json:"retry_limit"`
	SyncStatus            *string    `json:"sync_status"`
	HealthStatus          *string    `json:"health_status"`
	SyncRevision          *string    `json:"sync_revision"`
	OperationPhase        *string    `json:"operation_phase"`
	OperationMessage      *string    `json:"operation_message"`
	ResourceStatus        *string    `json:"resource_status"`
	Conditions            *string    `json:"conditions"`
	Labels                *string    `json:"labels"`
	Annotations           *string    `json:"annotations"`
	LastSyncedAt          *time.Time `json:"last_synced_at"`
}

type ApplicationList []*Application
type ApplicationIndex map[string]*Application

func (l ApplicationList) Index() ApplicationIndex {
	index := ApplicationIndex{}
	for _, o := range l {
		index[o.ID] = o
	}
	return index
}

func (d *Application) BeforeCreate(tx *gorm.DB) error {
	d.ID = api.NewID()
	return nil
}

type ApplicationPatchRequest struct {
	Name                  *string    `json:"name,omitempty"`
	SourceRepoUrl         *string    `json:"source_repo_url,omitempty"`
	SourceTargetRevision  *string    `json:"source_target_revision,omitempty"`
	SourcePath            *string    `json:"source_path,omitempty"`
	DestinationAmbientUrl *string    `json:"destination_ambient_url,omitempty"`
	DestinationProject    *string    `json:"destination_project,omitempty"`
	CredentialId          *string    `json:"credential_id,omitempty"`
	AutoSync              *bool      `json:"auto_sync,omitempty"`
	AutoPrune             *bool      `json:"auto_prune,omitempty"`
	SelfHeal              *bool      `json:"self_heal,omitempty"`
	SyncOptions           *string    `json:"sync_options,omitempty"`
	RetryLimit            *int       `json:"retry_limit,omitempty"`
	SyncStatus            *string    `json:"sync_status,omitempty"`
	HealthStatus          *string    `json:"health_status,omitempty"`
	SyncRevision          *string    `json:"sync_revision,omitempty"`
	OperationPhase        *string    `json:"operation_phase,omitempty"`
	OperationMessage      *string    `json:"operation_message,omitempty"`
	ResourceStatus        *string    `json:"resource_status,omitempty"`
	Conditions            *string    `json:"conditions,omitempty"`
	Labels                *string    `json:"labels,omitempty"`
	Annotations           *string    `json:"annotations,omitempty"`
	LastSyncedAt          *time.Time `json:"last_synced_at,omitempty"`
}
