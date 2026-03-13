package handlers

import (
	"fmt"
	"net/http"

	"ambient-code-public-api/types"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// GetSessionOutput handles GET /v1/sessions/:id/output
func GetSessionOutput(c *gin.Context) {
	project := GetProject(c)
	if !ValidateProjectName(project) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project name"})
		return
	}
	sessionID := c.Param("id")
	if !ValidateSessionID(sessionID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid session ID"})
		return
	}

	format := c.DefaultQuery("format", "transcript")
	if format != "transcript" && format != "compact" && format != "events" {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid format %q, must be one of: transcript, compact, events", format)})
		return
	}

	runIDFilter := c.Query("run_id")
	if runIDFilter != "" {
		if _, err := uuid.Parse(runIDFilter); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid run_id format, must be a valid UUID"})
			return
		}
	}

	events, statusCode, err := fetchSessionEvents(c, project, sessionID)
	if err != nil {
		if statusCode > 0 {
			c.JSON(statusCode, gin.H{"error": "Request failed"})
		} else {
			c.JSON(http.StatusBadGateway, gin.H{"error": "Backend unavailable"})
		}
		return
	}

	// Filter by run_id if provided
	if runIDFilter != "" {
		filtered := make([]map[string]interface{}, 0)
		for _, event := range events {
			if runID, _ := event["runId"].(string); runID == runIDFilter {
				filtered = append(filtered, event)
			}
		}
		events = filtered
	}

	switch format {
	case "events":
		c.JSON(http.StatusOK, types.EventsOutputResponse{
			SessionID: sessionID,
			Format:    "events",
			Events:    events,
		})
	case "compact":
		compacted := compactEvents(events)
		c.JSON(http.StatusOK, types.EventsOutputResponse{
			SessionID: sessionID,
			Format:    "compact",
			Events:    compacted,
		})
	case "transcript":
		messages := extractTranscript(events)
		c.JSON(http.StatusOK, types.TranscriptOutputResponse{
			SessionID: sessionID,
			Format:    "transcript",
			Messages:  messages,
		})
	}
}

// compactEvents merges consecutive TEXT_MESSAGE_CONTENT and TOOL_CALL_ARGS deltas.
func compactEvents(events []map[string]interface{}) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(events))

	for i := 0; i < len(events); i++ {
		event := events[i]
		eventType, _ := event["type"].(string)

		if eventType == "TEXT_MESSAGE_CONTENT" {
			messageID, _ := event["messageId"].(string)
			merged := copyEvent(event)
			delta, _ := merged["delta"].(string)

			for i+1 < len(events) {
				next := events[i+1]
				nextType, _ := next["type"].(string)
				nextMsgID, _ := next["messageId"].(string)
				if nextType == "TEXT_MESSAGE_CONTENT" && nextMsgID == messageID {
					nextDelta, _ := next["delta"].(string)
					delta += nextDelta
					i++
				} else {
					break
				}
			}
			merged["delta"] = delta
			result = append(result, merged)
		} else if eventType == "TOOL_CALL_ARGS" {
			toolCallID, _ := event["toolCallId"].(string)
			merged := copyEvent(event)
			delta, _ := merged["delta"].(string)

			for i+1 < len(events) {
				next := events[i+1]
				nextType, _ := next["type"].(string)
				nextTCID, _ := next["toolCallId"].(string)
				if nextType == "TOOL_CALL_ARGS" && nextTCID == toolCallID {
					nextDelta, _ := next["delta"].(string)
					delta += nextDelta
					i++
				} else {
					break
				}
			}
			merged["delta"] = delta
			result = append(result, merged)
		} else {
			result = append(result, event)
		}
	}

	return result
}

// extractTranscript finds the last MESSAGES_SNAPSHOT event and extracts messages.
func extractTranscript(events []map[string]interface{}) []types.TranscriptMessage {
	// Find last MESSAGES_SNAPSHOT event
	var snapshotMessages []interface{}
	for i := len(events) - 1; i >= 0; i-- {
		eventType, _ := events[i]["type"].(string)
		if eventType == "MESSAGES_SNAPSHOT" {
			if msgs, ok := events[i]["messages"].([]interface{}); ok {
				snapshotMessages = msgs
			}
			break
		}
	}

	if snapshotMessages == nil {
		return []types.TranscriptMessage{}
	}

	messages := make([]types.TranscriptMessage, 0, len(snapshotMessages))
	for _, raw := range snapshotMessages {
		msg, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}

		tm := types.TranscriptMessage{}
		if id, ok := msg["id"].(string); ok {
			tm.ID = id
		}
		if role, ok := msg["role"].(string); ok {
			tm.Role = role
		}
		if content, ok := msg["content"].(string); ok {
			tm.Content = content
		}
		if toolCallID, ok := msg["toolCallId"].(string); ok {
			tm.ToolCallID = toolCallID
		}
		if name, ok := msg["name"].(string); ok {
			tm.Name = name
		}
		if timestamp, ok := msg["timestamp"].(string); ok {
			tm.Timestamp = timestamp
		}

		// Extract tool calls if present
		if toolCalls, ok := msg["toolCalls"].([]interface{}); ok {
			for _, tcRaw := range toolCalls {
				tc, ok := tcRaw.(map[string]interface{})
				if !ok {
					continue
				}
				ttc := types.TranscriptToolCall{}
				if id, ok := tc["id"].(string); ok {
					ttc.ID = id
				}
				if name, ok := tc["name"].(string); ok {
					ttc.Name = name
				}
				if args, ok := tc["args"].(string); ok {
					ttc.Args = args
				}
				if result, ok := tc["result"].(string); ok {
					ttc.Result = result
				}
				if status, ok := tc["status"].(string); ok {
					ttc.Status = status
				}
				if duration, ok := tc["duration"]; ok {
					ttc.Duration = toInt64(duration)
				}
				tm.ToolCalls = append(tm.ToolCalls, ttc)
			}
		}

		messages = append(messages, tm)
	}

	return messages
}

// copyEvent creates a shallow copy of an event map.
func copyEvent(event map[string]interface{}) map[string]interface{} {
	copied := make(map[string]interface{}, len(event))
	for k, v := range event {
		copied[k] = v
	}
	return copied
}
