package tui

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
)

// ---------------------------------------------------------------------------
// EventColor
// ---------------------------------------------------------------------------

func TestEventColor(t *testing.T) {
	tests := []struct {
		eventType string
		want      lipgloss.Color
	}{
		{"user", lipgloss.Color("255")},
		{"assistant", lipgloss.Color("255")},
		{"tool_use", lipgloss.Color("240")},
		{"tool_result", lipgloss.Color("240")},
		{"system", lipgloss.Color("33")},
		{"error", lipgloss.Color("196")},
		{"unknown_type", lipgloss.Color("240")},
		{"", lipgloss.Color("240")},
	}
	for _, tt := range tests {
		t.Run(tt.eventType, func(t *testing.T) {
			got := EventColor(tt.eventType)
			if got != tt.want {
				t.Errorf("EventColor(%q) = %q, want %q", tt.eventType, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// EventSummary
// ---------------------------------------------------------------------------

func TestEventSummary_User(t *testing.T) {
	got := EventSummary("user", "Hello, world!")
	if got != "Hello, world!" {
		t.Errorf("got %q, want %q", got, "Hello, world!")
	}
}

func TestEventSummary_UserTruncation(t *testing.T) {
	long := make([]byte, 200)
	for i := range long {
		long[i] = 'a'
	}
	got := EventSummary("user", string(long))
	// truncatePayload uses byte slicing: s[:119] + "…" (3 bytes UTF-8) = 122 bytes.
	// Verify the rune count is at most 120.
	runes := []rune(got)
	if len(runes) > 120 {
		t.Errorf("expected truncation to <=120 runes, got %d", len(runes))
	}
	if len(runes) < 100 {
		t.Errorf("expected result near 120 runes, got only %d", len(runes))
	}
}

func TestEventSummary_Assistant(t *testing.T) {
	got := EventSummary("assistant", "I will help you.")
	if got != "I will help you." {
		t.Errorf("got %q, want %q", got, "I will help you.")
	}
}

func TestEventSummary_ToolUse_JSONPayload(t *testing.T) {
	payload := `{"name":"Read","arguments":{"file_path":"/tmp/foo.go"}}`
	got := EventSummary("tool_use", payload)
	// Should contain tool name.
	if got == "" {
		t.Fatal("expected non-empty summary")
	}
	if got != "Read file_path=/tmp/foo.go" {
		// Arguments are a map so iteration order is non-deterministic in general,
		// but with a single key it's stable.
		t.Errorf("got %q, want %q", got, "Read file_path=/tmp/foo.go")
	}
}

func TestEventSummary_ToolUse_NameOnly(t *testing.T) {
	payload := `{"name":"Bash"}`
	got := EventSummary("tool_use", payload)
	if got != "Bash" {
		t.Errorf("got %q, want %q", got, "Bash")
	}
}

func TestEventSummary_ToolUse_KVPayload(t *testing.T) {
	payload := `"name='Read'"`
	got := EventSummary("tool_use", payload)
	// Falls through to KV extraction via ExtractField.
	if got != "Read" {
		t.Errorf("got %q, want %q", got, "Read")
	}
}

func TestEventSummary_ToolResult_Success(t *testing.T) {
	payload := `{"content":"file contents here"}`
	got := EventSummary("tool_result", payload)
	want := "✓ 18 bytes"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestEventSummary_ToolResult_Error(t *testing.T) {
	payload := `{"content":"error details","is_error":true}`
	got := EventSummary("tool_result", payload)
	want := "✗ 13 bytes"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestEventSummary_ToolResult_Empty(t *testing.T) {
	got := EventSummary("tool_result", `{}`)
	want := "✓ 0 bytes"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestEventSummary_System(t *testing.T) {
	got := EventSummary("system", "System message")
	if got != "System message" {
		t.Errorf("got %q, want %q", got, "System message")
	}
}

func TestEventSummary_Error_JSONMessage(t *testing.T) {
	payload := `{"message":"connection refused"}`
	got := EventSummary("error", payload)
	want := "✗ connection refused"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestEventSummary_Error_PlainText(t *testing.T) {
	got := EventSummary("error", "something broke")
	want := "✗ something broke"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestEventSummary_Error_Empty(t *testing.T) {
	got := EventSummary("error", "")
	want := "✗ unknown error"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestEventSummary_TextMessageContent(t *testing.T) {
	payload := `{"delta":"hello world"}`
	got := EventSummary("TEXT_MESSAGE_CONTENT", payload)
	if got != "hello world" {
		t.Errorf("got %q, want %q", got, "hello world")
	}
}

func TestEventSummary_TextMessageContent_KV(t *testing.T) {
	payload := `"delta='hello world'"`
	got := EventSummary("TEXT_MESSAGE_CONTENT", payload)
	if got != "hello world" {
		t.Errorf("got %q, want %q", got, "hello world")
	}
}

func TestEventSummary_ReasoningMessageContent(t *testing.T) {
	payload := `{"delta":"thinking..."}`
	got := EventSummary("REASONING_MESSAGE_CONTENT", payload)
	if got != "thinking..." {
		t.Errorf("got %q, want %q", got, "thinking...")
	}
}

func TestEventSummary_ToolCallStart(t *testing.T) {
	payload := `{"tool_call_name":"Bash"}`
	got := EventSummary("TOOL_CALL_START", payload)
	if got != "⚙ Bash" {
		t.Errorf("got %q, want %q", got, "⚙ Bash")
	}
}

func TestEventSummary_ToolCallStart_ToolName(t *testing.T) {
	payload := `{"tool_name":"Read"}`
	got := EventSummary("TOOL_CALL_START", payload)
	if got != "⚙ Read" {
		t.Errorf("got %q, want %q", got, "⚙ Read")
	}
}

func TestEventSummary_ToolCallResult(t *testing.T) {
	payload := `{"content":"result data"}`
	got := EventSummary("TOOL_CALL_RESULT", payload)
	if got != "result data" {
		t.Errorf("got %q, want %q", got, "result data")
	}
}

func TestEventSummary_RunFinished(t *testing.T) {
	got := EventSummary("RUN_FINISHED", "")
	if got != "[done]" {
		t.Errorf("got %q, want %q", got, "[done]")
	}
}

func TestEventSummary_RunError(t *testing.T) {
	payload := `{"message":"out of memory"}`
	got := EventSummary("RUN_ERROR", payload)
	want := "✗ out of memory"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestEventSummary_RunError_KV(t *testing.T) {
	payload := `"message='out of memory'"`
	got := EventSummary("RUN_ERROR", payload)
	want := "✗ out of memory"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestEventSummary_TextMessageStart(t *testing.T) {
	got := EventSummary("TEXT_MESSAGE_START", "")
	if got != "…" {
		t.Errorf("got %q, want %q", got, "…")
	}
}

func TestEventSummary_SuppressedTypes(t *testing.T) {
	for _, et := range []string{"TEXT_MESSAGE_END", "TOOL_CALL_ARGS", "TOOL_CALL_END"} {
		got := EventSummary(et, "anything")
		if got != "" {
			t.Errorf("EventSummary(%q, ...) = %q, want empty", et, got)
		}
	}
}

func TestEventSummary_UnknownShortPayload(t *testing.T) {
	got := EventSummary("SOME_FUTURE_EVENT", "short payload")
	if got != "short payload" {
		t.Errorf("got %q, want %q", got, "short payload")
	}
}

func TestEventSummary_UnknownLongPayload(t *testing.T) {
	long := make([]byte, 200)
	for i := range long {
		long[i] = 'x'
	}
	got := EventSummary("SOME_FUTURE_EVENT", string(long))
	if got != "" {
		t.Errorf("got %q, want empty for long unknown payload", got)
	}
}

// ---------------------------------------------------------------------------
// ParsePayload
// ---------------------------------------------------------------------------

func TestParsePayload_ValidJSON(t *testing.T) {
	result := ParsePayload(`{"name":"Read","count":42}`)
	if result["name"] != "Read" {
		t.Errorf("name = %v, want Read", result["name"])
	}
	// JSON numbers are float64.
	if result["count"] != float64(42) {
		t.Errorf("count = %v, want 42", result["count"])
	}
}

func TestParsePayload_InvalidJSON(t *testing.T) {
	result := ParsePayload("not json at all")
	raw, ok := result["raw"]
	if !ok {
		t.Fatal("expected 'raw' key for invalid JSON")
	}
	if raw != "not json at all" {
		t.Errorf("raw = %q, want %q", raw, "not json at all")
	}
}

func TestParsePayload_Empty(t *testing.T) {
	result := ParsePayload("")
	if len(result) != 0 {
		t.Errorf("expected empty map for empty input, got %v", result)
	}
}

func TestParsePayload_JSONArray(t *testing.T) {
	result := ParsePayload(`[1,2,3]`)
	// Not an object, should fall back to raw.
	if _, ok := result["raw"]; !ok {
		t.Error("expected 'raw' key for JSON array")
	}
}

func TestParsePayload_JSONString(t *testing.T) {
	result := ParsePayload(`"just a string"`)
	if _, ok := result["raw"]; !ok {
		t.Error("expected 'raw' key for JSON string")
	}
}

func TestParsePayload_Nested(t *testing.T) {
	result := ParsePayload(`{"outer":{"inner":"value"}}`)
	outer, ok := result["outer"].(map[string]any)
	if !ok {
		t.Fatal("expected nested object for 'outer'")
	}
	if outer["inner"] != "value" {
		t.Errorf("inner = %v, want value", outer["inner"])
	}
}

// ---------------------------------------------------------------------------
// ExtractField
// ---------------------------------------------------------------------------

func TestExtractField_JSONObject(t *testing.T) {
	payload := `{"delta":"hello","seq":5}`
	if got := ExtractField(payload, "delta"); got != "hello" {
		t.Errorf("got %q, want %q", got, "hello")
	}
	if got := ExtractField(payload, "seq"); got != "5" {
		t.Errorf("got %q, want %q", got, "5")
	}
}

func TestExtractField_JSONMissing(t *testing.T) {
	payload := `{"delta":"hello"}`
	if got := ExtractField(payload, "missing"); got != "" {
		t.Errorf("got %q, want empty", got)
	}
}

func TestExtractField_JSONNested(t *testing.T) {
	payload := `{"args":{"file":"/tmp/foo"}}`
	got := ExtractField(payload, "args")
	// Should return JSON representation of the nested object.
	if got != `{"file":"/tmp/foo"}` {
		t.Errorf("got %q, want %q", got, `{"file":"/tmp/foo"}`)
	}
}

func TestExtractField_JSONNull(t *testing.T) {
	payload := `{"value":null}`
	if got := ExtractField(payload, "value"); got != "" {
		t.Errorf("got %q, want empty for null", got)
	}
}

func TestExtractField_JSONBool(t *testing.T) {
	payload := `{"is_error":true}`
	if got := ExtractField(payload, "is_error"); got != "true" {
		t.Errorf("got %q, want %q", got, "true")
	}
}

func TestExtractField_KVFormat(t *testing.T) {
	payload := `delta='hello world'`
	if got := ExtractField(payload, "delta"); got != "hello world" {
		t.Errorf("got %q, want %q", got, "hello world")
	}
}

func TestExtractField_KVFormatEscaped(t *testing.T) {
	payload := `msg='it\'s fine'`
	if got := ExtractField(payload, "msg"); got != "it's fine" {
		t.Errorf("got %q, want %q", got, "it's fine")
	}
}

func TestExtractField_KVFormatJSONWrapped(t *testing.T) {
	// Payload is a JSON string containing KV format.
	payload := `"delta='hello'"`
	if got := ExtractField(payload, "delta"); got != "hello" {
		t.Errorf("got %q, want %q", got, "hello")
	}
}

func TestExtractField_KVMissing(t *testing.T) {
	payload := `name='Read'`
	if got := ExtractField(payload, "missing"); got != "" {
		t.Errorf("got %q, want empty", got)
	}
}

func TestExtractField_EmptyPayload(t *testing.T) {
	if got := ExtractField("", "key"); got != "" {
		t.Errorf("got %q, want empty", got)
	}
}

func TestExtractField_JSONFloat(t *testing.T) {
	payload := `{"ratio":3.14}`
	if got := ExtractField(payload, "ratio"); got != "3.14" {
		t.Errorf("got %q, want %q", got, "3.14")
	}
}
