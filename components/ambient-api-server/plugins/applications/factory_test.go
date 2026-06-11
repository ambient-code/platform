package applications_test

import (
	"context"
	"fmt"
	"time"

	"github.com/ambient-code/platform/components/ambient-api-server/plugins/applications"
	"github.com/openshift-online/rh-trex-ai/pkg/environments"
)

func newApplication(id string) (*applications.Application, error) {
	applicationService := applications.Service(&environments.Environment().Services)

	application := &applications.Application{
		Name:                  "test-name",
		SourceRepoUrl:         "test-source_repo_url",
		SourceTargetRevision:  stringPtr("test-source_target_revision"),
		SourcePath:            "test-source_path",
		DestinationAmbientUrl: stringPtr("test-destination_ambient_url"),
		DestinationProject:    "test-destination_project",
		CredentialId:          stringPtr("test-credential_id"),
		AutoSync:              boolPtr(true),
		AutoPrune:             boolPtr(true),
		SelfHeal:              boolPtr(true),
		SyncOptions:           stringPtr("test-sync_options"),
		RetryLimit:            intPtr(42),
		SyncStatus:            stringPtr("test-sync_status"),
		HealthStatus:          stringPtr("test-health_status"),
		SyncRevision:          stringPtr("test-sync_revision"),
		OperationPhase:        stringPtr("test-operation_phase"),
		OperationMessage:      stringPtr("test-operation_message"),
		ResourceStatus:        stringPtr("test-resource_status"),
		Conditions:            stringPtr("test-conditions"),
		Labels:                stringPtr("test-labels"),
		Annotations:           stringPtr("test-annotations"),
		LastSyncedAt:          timePtr(time.Now()),
	}

	sub, err := applicationService.Create(context.Background(), application)
	if err != nil {
		return nil, err
	}

	return sub, nil
}

func newApplicationList(namePrefix string, count int) ([]*applications.Application, error) {
	var items []*applications.Application
	for i := 1; i <= count; i++ {
		name := fmt.Sprintf("%s_%d", namePrefix, i)
		c, err := newApplication(name)
		if err != nil {
			return nil, err
		}
		items = append(items, c)
	}
	return items, nil
}
func stringPtr(s string) *string     { return &s }
func intPtr(i int) *int              { return &i }
func boolPtr(b bool) *bool           { return &b }
func timePtr(t time.Time) *time.Time { return &t }
