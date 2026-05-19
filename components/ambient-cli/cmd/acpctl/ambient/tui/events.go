package tui

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// EventColor returns the lipgloss color for a semantic event type.
//
// Mapping follows the TUI spec's "Event Type Rendering" table:
//
//	user        -> white (255)
//	assistant   -> white (255)
//	tool_use    -> dim   (240)
//	tool_result -> dim   (240)
//	system      -> yellow (33)
//	error       -> red   (196)
func EventColor(eventType string) lipgloss.Color {
	switch eventType {
	case "user":
		return colorWhite // 255
	case "assistant":
		return colorWhite // 255 — assistant text is primary content
	case "tool_use":
		return colorDim // 240
	case "tool_result":
		return colorDim // 240
	case "system":
		return colorYellow // 33
	case "error":
		return colorRed // 31
	default:
		return colorDim // 240
	}
}

// PhaseColor returns the display color for a session phase.
//
//	pending              -> yellow (33)
//	running / active     -> orange (214)
//	succeeded / completed -> dim   (240)
//	failed               -> red    (31)
//	cancelled            -> dim    (240)
func PhaseColor(phase string) lipgloss.Color {
	switch strings.ToLower(phase) {
	case "pending":
		return colorYellow // 33
	case "running", "active":
		return colorOrange // 214
	case "succeeded", "completed":
		return colorDim // 240
	case "failed":
		return colorRed // 31
	case "cancelled":
		return colorDim // 240
	default:
		return colorDim // 240
	}
}

// EventSummary returns a one-line display summary for an AG-UI event.
//
// Behaviour is extracted from the existing tileDisplayPayload logic:
//
//	user                    -> payload text, truncated to 120 chars
//	assistant               -> payload text, truncated to 120 chars
//	tool_use                -> tool name + first argument, truncated
//	tool_result             -> checkmark/cross + content size
//	system                  -> payload text, truncated to 120 chars
//	error                   -> cross + error message
//	TEXT_MESSAGE_CONTENT    -> delta field from payload
//	REASONING_MESSAGE_CONTENT -> delta field from payload
//	TOOL_CALL_START         -> gear icon + tool name
//	TOOL_CALL_RESULT        -> content field from payload
//	RUN_FINISHED            -> "[done]"
//	RUN_ERROR               -> cross + error message
//	TEXT_MESSAGE_START       -> ellipsis
//	TEXT_MESSAGE_END, TOOL_CALL_ARGS, TOOL_CALL_END -> empty (suppressed)
func EventSummary(eventType string, payload string) string {
	switch eventType {
	case "user":
		return truncatePayload(payload, 120)

	case "assistant":
		return truncatePayload(payload, 120)

	case "tool_use":
		parsed := ParsePayload(payload)
		name, _ := parsed["name"].(string)
		if name == "" {
			name = ExtractField(payload, "name")
		}
		if name == "" {
			return truncatePayload(payload, 120)
		}
		// Include first argument if available.
		firstArg := ""
		if args, ok := parsed["arguments"].(map[string]any); ok {
			for k, v := range args {
				firstArg = fmt.Sprintf("%s=%v", k, v)
				break
			}
		}
		if firstArg == "" {
			if a := ExtractField(payload, "input"); a != "" {
				firstArg = truncatePayload(a, 60)
			}
		}
		if firstArg != "" {
			return name + " " + truncatePayload(firstArg, 80)
		}
		return name

	case "tool_result":
		parsed := ParsePayload(payload)
		content, _ := parsed["content"].(string)
		if content == "" {
			content = ExtractField(payload, "content")
		}
		isError := false
		if e, ok := parsed["is_error"].(bool); ok {
			isError = e
		}
		if isError {
			size := len(content)
			return fmt.Sprintf("✗ %d bytes", size)
		}
		size := len(content)
		return fmt.Sprintf("✓ %d bytes", size)

	case "system":
		return truncatePayload(payload, 120)

	case "error":
		parsed := ParsePayload(payload)
		if errMsg, ok := parsed["message"].(string); ok && errMsg != "" {
			return "✗ " + truncatePayload(errMsg, 120)
		}
		if errMsg := ExtractField(payload, "message"); errMsg != "" {
			return "✗ " + truncatePayload(errMsg, 120)
		}
		if payload != "" {
			return "✗ " + truncatePayload(payload, 120)
		}
		return "✗ unknown error"

	// AG-UI wire event types (carried forward from tileDisplayPayload).
	case "TEXT_MESSAGE_CONTENT", "REASONING_MESSAGE_CONTENT":
		if d := ExtractField(payload, "delta"); d != "" {
			return d
		}
		return ""

	case "TOOL_CALL_START":
		if name := ExtractField(payload, "tool_call_name"); name != "" {
			return "⚙ " + name
		}
		if name := ExtractField(payload, "tool_name"); name != "" {
			return "⚙ " + name
		}
		return ""

	case "TOOL_CALL_RESULT":
		if c := ExtractField(payload, "content"); c != "" {
			return c
		}
		return ""

	case "RUN_FINISHED":
		return "[done]"

	case "RUN_ERROR":
		if errMsg := ExtractField(payload, "message"); errMsg != "" {
			return "✗ " + errMsg
		}
		return "✗ error"

	case "TEXT_MESSAGE_START":
		return "…"

	case "TEXT_MESSAGE_END", "TOOL_CALL_ARGS", "TOOL_CALL_END":
		return ""
	}

	// Fallback: show raw payload if short enough.
	if payload != "" && len(payload) <= 120 {
		return payload
	}
	return ""
}

// ParsePayload safely parses a JSON payload string into a map.
// If the payload is not valid JSON, returns a map with a single "raw" key
// containing the original string.
// Returns an empty map for empty input.
func ParsePayload(payload string) map[string]any {
	if payload == "" {
		return map[string]any{}
	}

	var result map[string]any
	if err := json.Unmarshal([]byte(payload), &result); err == nil {
		return result
	}

	// Payload is not a JSON object. Return it under "raw".
	return map[string]any{
		"raw": payload,
	}
}

// ExtractField extracts a specific field value from a payload string.
//
// It first attempts JSON object parsing (for {"key": "value"} payloads).
// If that fails, it falls back to the KV format used by the AG-UI runner:
// key='value' with backslash-escaped single quotes inside.
//
// Returns an empty string if the field is not found.
func ExtractField(payload string, key string) string {
	// Try JSON object parse first.
	var obj map[string]any
	if err := json.Unmarshal([]byte(payload), &obj); err == nil {
		if v, ok := obj[key]; ok {
			switch val := v.(type) {
			case string:
				return val
			case float64:
				// Preserve integer formatting when possible.
				if val == float64(int64(val)) {
					return fmt.Sprintf("%d", int64(val))
				}
				return fmt.Sprintf("%g", val)
			case bool:
				return fmt.Sprintf("%t", val)
			case nil:
				return ""
			default:
				b, _ := json.Marshal(val)
				return string(b)
			}
		}
		return ""
	}

	// Fall back to the existing extractKVField logic: key='value' format.
	// The payload may be a JSON-encoded string (double-quoted), so unwrap first.
	var raw string
	if err := json.Unmarshal([]byte(payload), &raw); err == nil {
		payload = raw
	}

	needle := key + "='"
	idx := strings.Index(payload, needle)
	if idx < 0 {
		return ""
	}
	start := idx + len(needle)
	var sb strings.Builder
	for i := start; i < len(payload); i++ {
		if payload[i] == '\'' && (i == start || payload[i-1] != '\\') {
			break
		}
		sb.WriteByte(payload[i])
	}
	return strings.ReplaceAll(sb.String(), `\'`, `'`)
}

// truncatePayload trims whitespace and truncates a string to max length.
func truncatePayload(s string, max int) string {
	s = strings.TrimSpace(s)
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	if max <= 1 {
		return "…"
	}
	return string(r[:max-1]) + "…"
}
