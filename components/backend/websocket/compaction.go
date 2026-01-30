package websocket

import (
	"ambient-code-backend/types"
	"log"

	"github.com/google/uuid"
)

// MessageCompactor compacts AG-UI events into message snapshots
// Per AG-UI spec: https://docs.ag-ui.com/concepts/serialization
type MessageCompactor struct {
	messages        []types.Message
	currentMessage  *types.Message
	activeToolCalls map[string]*ActiveToolCall // toolId -> tool state
	hiddenMessages  map[string]bool            // messageId -> hidden flag
}

// ActiveToolCall tracks an in-progress tool call
type ActiveToolCall struct {
	ID              string
	Name            string
	Args            string // Accumulated from TOOL_CALL_ARGS deltas
	ParentToolUseID string
	Status          string
}

// NewMessageCompactor creates a new message compactor
func NewMessageCompactor() *MessageCompactor {
	return &MessageCompactor{
		messages:        make([]types.Message, 0),
		activeToolCalls: make(map[string]*ActiveToolCall),
		hiddenMessages:  make(map[string]bool),
	}
}

// HandleEvent processes a single AG-UI event and updates compacted state
func (c *MessageCompactor) HandleEvent(event map[string]interface{}) {
	eventType, _ := event["type"].(string)

	switch eventType {
	case types.EventTypeTextMessageStart:
		c.handleTextMessageStart(event)
	case types.EventTypeTextMessageContent:
		c.handleTextMessageContent(event)
	case types.EventTypeTextMessageEnd:
		c.handleTextMessageEnd(event)
	case types.EventTypeToolCallStart:
		c.handleToolCallStart(event)
	case types.EventTypeToolCallArgs:
		c.handleToolCallArgs(event)
	case types.EventTypeToolCallEnd:
		c.handleToolCallEnd(event)
	case types.EventTypeRaw:
		c.handleRawEvent(event)
	case types.EventTypeMessagesSnapshot:
		c.handleMessagesSnapshot(event)
	case types.EventTypeRunStarted, types.EventTypeRunFinished, types.EventTypeRunError:
		// Lifecycle events - skip, don't affect message compaction
	case types.EventTypeStepStarted, types.EventTypeStepFinished:
		// Step events - skip, don't affect message compaction
	case types.EventTypeStateSnapshot, types.EventTypStateDelta:
		// State events - skip, don't affect message compaction
	case types.EventTypeActivitySnapshot, types.EventTypeActivityDelta:
		// Activity events - skip, don't affect message compaction
	default:
		log.Printf("Compaction: WARNING - Unhandled event type: %s", eventType)
	}
}

// GetMessages returns the compacted messages (excluding hidden ones)
func (c *MessageCompactor) GetMessages() []types.Message {
	// Flush any active message
	if c.currentMessage != nil {
		c.messages = append(c.messages, *c.currentMessage)
		c.currentMessage = nil
	}

	// DO NOT include in-progress tools in snapshots!
	// Snapshots should only contain COMPLETED runs with finished tool calls.
	// In-progress tools will be streamed as raw events from the active run.
	//
	// If we included "running" status tools here, they would duplicate when
	// the active run's TOOL_CALL_END events are replayed.
	if len(c.activeToolCalls) > 0 {
		// Clear activeToolCalls - don't include them in snapshot
		c.activeToolCalls = make(map[string]*ActiveToolCall)
	}

	// Filter out hidden messages (auto-sent initial/workflow prompts)
	visibleMessages := make([]types.Message, 0, len(c.messages))
	hiddenCount := 0
	for _, msg := range c.messages {
		if c.hiddenMessages[msg.ID] {
			hiddenCount++
			continue
		}
		visibleMessages = append(visibleMessages, msg)
	}

	return visibleMessages
}

// Event Handlers

func (c *MessageCompactor) handleTextMessageStart(event map[string]interface{}) {
	// Flush previous message if any
	if c.currentMessage != nil {
		c.messages = append(c.messages, *c.currentMessage)
	}

	// Handle both camelCase and snake_case
	messageID, _ := event["messageId"].(string)
	if messageID == "" {
		messageID, _ = event["message_id"].(string)
	}
	role, _ := event["role"].(string)
	if role == "" {
		role = types.RoleAssistant
	}
	// Preserve timestamp from the event
	timestamp, _ := event["timestamp"].(string)

	c.currentMessage = &types.Message{
		ID:        messageID,
		Role:      role,
		Content:   "",
		Timestamp: timestamp,
	}
}

func (c *MessageCompactor) handleTextMessageContent(event map[string]interface{}) {
	if c.currentMessage == nil {
		return
	}

	delta, _ := event["delta"].(string)
	c.currentMessage.Content += delta
}

func (c *MessageCompactor) handleTextMessageEnd(event map[string]interface{}) {
	if c.currentMessage != nil {
		// User messages never have tool calls - flush immediately
		// Assistant messages might have tool calls - keep open
		// We'll flush when a new TEXT_MESSAGE_START arrives or at the end of compaction
		if c.currentMessage.Role == types.RoleUser {
			c.messages = append(c.messages, *c.currentMessage)
			c.currentMessage = nil
		}
	}
}

func (c *MessageCompactor) handleToolCallStart(event map[string]interface{}) {
	// Handle both camelCase (TypeScript) and snake_case (Python ag_ui.core)
	toolID, _ := event["toolCallId"].(string)
	if toolID == "" {
		toolID, _ = event["tool_call_id"].(string)
	}
	toolName, _ := event["toolCallName"].(string)
	if toolName == "" {
		toolName, _ = event["tool_call_name"].(string)
	}

	// Try multiple field names for parent tool ID
	parentToolUseID, _ := event["parentToolUseId"].(string)
	if parentToolUseID == "" {
		parentToolUseID, _ = event["parentToolUseID"].(string)
	}
	if parentToolUseID == "" {
		parentToolUseID, _ = event["parent_tool_call_id"].(string)
	}

	if toolID != "" {
		c.activeToolCalls[toolID] = &ActiveToolCall{
			ID:              toolID,
			Name:            toolName,
			Args:            "",
			ParentToolUseID: parentToolUseID,
			Status:          "running",
		}
	}
}

func (c *MessageCompactor) handleToolCallArgs(event map[string]interface{}) {
	// Handle both camelCase and snake_case
	toolID, _ := event["toolCallId"].(string)
	if toolID == "" {
		toolID, _ = event["tool_call_id"].(string)
	}
	delta, _ := event["delta"].(string)

	if toolID == "" {
		return
	}

	if active, ok := c.activeToolCalls[toolID]; ok {
		active.Args += delta
	}
}

func (c *MessageCompactor) handleToolCallEnd(event map[string]interface{}) {
	// Handle both camelCase and snake_case
	toolID, _ := event["toolCallId"].(string)
	if toolID == "" {
		toolID, _ = event["tool_call_id"].(string)
	}
	result, _ := event["result"].(string)
	errorStr, _ := event["error"].(string)

	if toolID == "" {
		return
	}

	active, ok := c.activeToolCalls[toolID]
	if !ok {
		return
	}

	// Create completed tool call
	tc := types.ToolCall{
		ID:              active.ID,
		Name:            active.Name,
		Args:            active.Args,
		Type:            "function",
		ParentToolUseID: active.ParentToolUseID,
		Result:          result,
		Status:          "completed",
	}
	if errorStr != "" {
		tc.Error = errorStr
		tc.Status = "error"
	}

	// Preserve timestamp from the event
	timestamp, _ := event["timestamp"].(string)

	// Add to message
	// Check if we need to create a new message or add to current
	if c.currentMessage != nil && c.currentMessage.Role == types.RoleAssistant {
		// Add to current message
		c.currentMessage.ToolCalls = append(c.currentMessage.ToolCalls, tc)
		// Update timestamp if not already set
		if c.currentMessage.Timestamp == "" && timestamp != "" {
			c.currentMessage.Timestamp = timestamp
		}
	} else {
		// Create new message for this tool call
		c.messages = append(c.messages, types.Message{
			ID:        uuid.New().String(),
			Role:      types.RoleAssistant,
			ToolCalls: []types.ToolCall{tc},
			Timestamp: timestamp,
		})
	}

	// Remove from active
	delete(c.activeToolCalls, toolID)
}

func (c *MessageCompactor) handleRawEvent(event map[string]interface{}) {
	// Check for both "data" and "event" fields (AG-UI uses "event")
	var data map[string]interface{}
	if d, ok := event["event"].(map[string]interface{}); ok {
		data = d
	} else if d, ok := event["data"].(map[string]interface{}); ok {
		data = d
	} else {
		return
	}

	// Handle message_metadata events (for hiding auto-sent prompts)
	if msgType, _ := data["type"].(string); msgType == "message_metadata" {
		if hidden, _ := data["hidden"].(bool); hidden {
			if messageID, ok := data["messageId"].(string); ok {
				c.hiddenMessages[messageID] = true
			}
		}
		return
	}

	role, _ := data["role"].(string)
	if role == "" {
		return
	}

	// Flush current message
	if c.currentMessage != nil {
		c.messages = append(c.messages, *c.currentMessage)
		c.currentMessage = nil
	}

	// Add raw message
	msg := types.Message{Role: role}
	if id, ok := data["id"].(string); ok {
		msg.ID = id
	}
	if content, ok := data["content"].(string); ok {
		msg.Content = content
	}
	if timestamp, ok := data["timestamp"].(string); ok {
		msg.Timestamp = timestamp
	}

	c.messages = append(c.messages, msg)
}

func (c *MessageCompactor) handleMessagesSnapshot(event map[string]interface{}) {
	// If runner sends MESSAGES_SNAPSHOT, use it directly (overrides compaction)
	msgs, ok := event["messages"].([]interface{})
	if !ok {
		return
	}

	// Replace all messages with snapshot
	c.messages = make([]types.Message, 0, len(msgs))
	c.currentMessage = nil

	for _, m := range msgs {
		msgMap, ok := m.(map[string]interface{})
		if !ok {
			continue
		}

		msg := types.Message{}
		if id, ok := msgMap["id"].(string); ok {
			msg.ID = id
		}
		if role, ok := msgMap["role"].(string); ok {
			msg.Role = role
		}
		if content, ok := msgMap["content"].(string); ok {
			msg.Content = content
		}
		if timestamp, ok := msgMap["timestamp"].(string); ok {
			msg.Timestamp = timestamp
		}

		// Extract toolCalls array
		if toolCalls, ok := msgMap["toolCalls"].([]interface{}); ok {
			msg.ToolCalls = make([]types.ToolCall, 0, len(toolCalls))
			for _, tc := range toolCalls {
				tcMap, ok := tc.(map[string]interface{})
				if !ok {
					continue
				}

				toolCall := types.ToolCall{}
				if id, ok := tcMap["id"].(string); ok {
					toolCall.ID = id
				}
				if name, ok := tcMap["name"].(string); ok {
					toolCall.Name = name
				}
				if args, ok := tcMap["args"].(string); ok {
					toolCall.Args = args
				}
				if tcType, ok := tcMap["type"].(string); ok {
					toolCall.Type = tcType
				}
				if parentID, ok := tcMap["parentToolUseId"].(string); ok {
					toolCall.ParentToolUseID = parentID
				}
				if result, ok := tcMap["result"].(string); ok {
					toolCall.Result = result
				}
				if status, ok := tcMap["status"].(string); ok {
					toolCall.Status = status
				}
				if errorStr, ok := tcMap["error"].(string); ok {
					toolCall.Error = errorStr
				}

				msg.ToolCalls = append(msg.ToolCalls, toolCall)
			}
		}

		c.messages = append(c.messages, msg)
	}

}

// CompactEvents is the main entry point for event compaction
func CompactEvents(events []map[string]interface{}) []types.Message {

	// Count event types to help debug
	eventTypeCounts := make(map[string]int)
	for _, event := range events {
		eventType, _ := event["type"].(string)
		eventTypeCounts[eventType]++
	}

	compactor := NewMessageCompactor()

	for _, event := range events {
		compactor.HandleEvent(event)
	}

	messages := compactor.GetMessages()

	return messages
}
