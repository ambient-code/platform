package websocket

import (
	"testing"

	"ambient-code-backend/types"
)

func TestIsActivityEvent(t *testing.T) {
	activityEvents := []struct {
		name      string
		eventType string
	}{
		{"RUN_STARTED", types.EventTypeRunStarted},
		{"TEXT_MESSAGE_START", types.EventTypeTextMessageStart},
		{"TEXT_MESSAGE_CONTENT", types.EventTypeTextMessageContent},
		{"TOOL_CALL_START", types.EventTypeToolCallStart},
	}

	for _, tc := range activityEvents {
		t.Run(tc.name+" is activity", func(t *testing.T) {
			if !isActivityEvent(tc.eventType) {
				t.Errorf("expected %s to be an activity event", tc.name)
			}
		})
	}

	nonActivityEvents := []struct {
		name      string
		eventType string
	}{
		{"RUN_FINISHED", types.EventTypeRunFinished},
		{"RUN_ERROR", types.EventTypeRunError},
		{"STEP_STARTED", types.EventTypeStepStarted},
		{"STEP_FINISHED", types.EventTypeStepFinished},
		{"TEXT_MESSAGE_END", types.EventTypeTextMessageEnd},
		{"TOOL_CALL_ARGS", types.EventTypeToolCallArgs},
		{"TOOL_CALL_END", types.EventTypeToolCallEnd},
		{"STATE_SNAPSHOT", types.EventTypeStateSnapshot},
		{"STATE_DELTA", types.EventTypeStateDelta},
		{"MESSAGES_SNAPSHOT", types.EventTypeMessagesSnapshot},
		{"RAW", types.EventTypeRaw},
		{"META", types.EventTypeMeta},
		{"empty string", ""},
		{"unknown event", "UNKNOWN_EVENT"},
	}

	for _, tc := range nonActivityEvents {
		t.Run(tc.name+" is not activity", func(t *testing.T) {
			if isActivityEvent(tc.eventType) {
				t.Errorf("expected %s to NOT be an activity event", tc.name)
			}
		})
	}
}
