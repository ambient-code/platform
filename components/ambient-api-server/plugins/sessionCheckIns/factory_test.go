package sessionCheckIns_test

import (
	"context"
	"fmt"

	"github.com/ambient-code/platform/components/ambient-api-server/plugins/sessionCheckIns"
	"github.com/openshift-online/rh-trex-ai/pkg/environments"
)

func newSessionCheckIn(id string) (*sessionCheckIns.SessionCheckIn, error) {
	sessionCheckInService := sessionCheckIns.Service(&environments.Environment().Services)

	sessionCheckIn := &sessionCheckIns.SessionCheckIn{
		SessionId: "test-session_id",
		AgentId:   "test-agent_id",
		Summary:   stringPtr("test-summary"),
		Branch:    stringPtr("test-branch"),
		Worktree:  stringPtr("test-worktree"),
		Pr:        stringPtr("test-pr"),
		Phase:     stringPtr("test-phase"),
		TestCount: intPtr(42),
		NextSteps: stringPtr("test-next_steps"),
	}

	sub, err := sessionCheckInService.Create(context.Background(), sessionCheckIn)
	if err != nil {
		return nil, err
	}

	return sub, nil
}

func newSessionCheckInList(namePrefix string, count int) ([]*sessionCheckIns.SessionCheckIn, error) {
	var items []*sessionCheckIns.SessionCheckIn
	for i := 1; i <= count; i++ {
		name := fmt.Sprintf("%s_%d", namePrefix, i)
		c, err := newSessionCheckIn(name)
		if err != nil {
			return nil, err
		}
		items = append(items, c)
	}
	return items, nil
}
func stringPtr(s string) *string { return &s }
func intPtr(i int) *int          { return &i }
