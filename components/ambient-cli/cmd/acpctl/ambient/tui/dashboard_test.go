package tui

import (
	"strings"
	"testing"
	"time"

	sdktypes "github.com/ambient-code/platform/components/ambient-sdk/go-sdk/types"
)

func msg(eventType, payload string, createdAt *time.Time) sdktypes.SessionMessage {
	m := sdktypes.SessionMessage{}
	m.EventType = eventType
	m.Payload = payload
	m.CreatedAt = createdAt
	return m
}

func ptr(t time.Time) *time.Time { return &t }

func TestRenderTileMessages_NilCreatedAt(t *testing.T) {
	msgs := []sdktypes.SessionMessage{
		msg("user", "hello from user", nil),
	}
	lines := renderTileMessages(msgs, 100, 10)
	found := false
	for _, l := range lines {
		if strings.Contains(l, "hello from user") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'hello from user' in rendered lines, got: %v", lines)
	}
}

func TestRenderTileMessages_WithCreatedAt(t *testing.T) {
	ts := time.Date(2026, 1, 1, 13, 45, 0, 0, time.UTC)
	msgs := []sdktypes.SessionMessage{
		msg("user", "timestamped message", ptr(ts)),
	}
	lines := renderTileMessages(msgs, 100, 10)
	found := false
	for _, l := range lines {
		if strings.Contains(l, "13:45:00") && strings.Contains(l, "timestamped message") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected timestamp and payload in rendered lines, got: %v", lines)
	}
}

func TestRenderTileMessages_FilteredEventTypes(t *testing.T) {
	msgs := []sdktypes.SessionMessage{
		msg("TEXT_MESSAGE_END", "should be hidden", nil),
		msg("TOOL_CALL_ARGS", "also hidden", nil),
		msg("user", "visible", nil),
	}
	lines := renderTileMessages(msgs, 100, 10)
	for _, l := range lines {
		if strings.Contains(l, "should be hidden") || strings.Contains(l, "also hidden") {
			t.Errorf("filtered event type appeared in output: %q", l)
		}
	}
	found := false
	for _, l := range lines {
		if strings.Contains(l, "visible") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'visible' in rendered lines, got: %v", lines)
	}
}

func TestRenderTileMessages_MaxLines(t *testing.T) {
	var msgs []sdktypes.SessionMessage
	for i := 0; i < 20; i++ {
		msgs = append(msgs, msg("user", "msg", nil))
	}
	lines := renderTileMessages(msgs, 100, 5)
	if len(lines) != 5 {
		t.Errorf("expected 5 lines (maxLines), got %d", len(lines))
	}
}

func TestTileDisplayPayload_UserEvent(t *testing.T) {
	m := msg("user", "hello", nil)
	got := tileDisplayPayload(m)
	if got != "hello" {
		t.Errorf("expected 'hello', got %q", got)
	}
}

func TestTileDisplayPayload_RunFinished(t *testing.T) {
	m := msg("RUN_FINISHED", "", nil)
	got := tileDisplayPayload(m)
	if got != "[done]" {
		t.Errorf("expected '[done]', got %q", got)
	}
}

func TestTileDisplayPayload_EmptyFiltered(t *testing.T) {
	for _, et := range []string{"TEXT_MESSAGE_END", "TOOL_CALL_ARGS", "TOOL_CALL_END"} {
		m := msg(et, "anything", nil)
		got := tileDisplayPayload(m)
		if got != "" {
			t.Errorf("event type %q should return empty, got %q", et, got)
		}
	}
}
