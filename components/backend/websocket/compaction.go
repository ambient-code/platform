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
		log.Printf("Compaction: Skipping lifecycle event %s", eventType)
	case types.EventTypeStepStarted, types.EventTypeStepFinished:
		// Step events - skip, don't affect message compaction
		log.Printf("Compaction: Skipping step event %s", eventType)
	case types.EventTypeStateSnapshot, types.EventTypStateDelta:
		// State events - skip, don't affect message compaction
		log.Printf("Compaction: Skipping state event %s", eventType)
	case types.EventTypeActivitySnapshot, types.EventTypeActivityDelta:
		// Activity events - skip, don't affect message compaction
		log.Printf("Compaction: Skipping activity event %s", eventType)
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
		log.Printf("Compaction: WARNING - %d tool calls never completed (missing TOOL_CALL_END)", len(c.activeToolCalls))
		log.Printf("Compaction: This indicates events from an active/incomplete run were included in compaction")
		log.Printf("Compaction: Discarding incomplete tool calls to prevent duplicates")
		// Clear activeToolCalls - don't include them in snapshot
		c.activeToolCalls = make(map[string]*ActiveToolCall)
	}

	// Filter out hidden messages (auto-sent initial/workflow prompts)
	visibleMessages := make([]types.Message, 0, len(c.messages))
	hiddenCount := 0
	for _, msg := range c.messages {
		if c.hiddenMessages[msg.ID] {
			log.Printf("Compaction: Filtering hidden message %s from snapshot", msg.ID[:8])
			hiddenCount++
			continue
		}
		visibleMessages = append(visibleMessages, msg)
	}

	if hiddenCount > 0 {
		log.Printf("Compaction: Filtered %d hidden messages, returning %d visible", hiddenCount, len(visibleMessages))
	}

	return visibleMessages
}

// Event Handlers

func (c *MessageCompactor) handleTextMessageStart(event map[string]interface{}) {
	// Flush previous message if any
	if c.currentMessage != nil {
		log.Printf("Compaction: Flushing previous message (id=%s, content=%d chars)", c.currentMessage.ID, len(c.currentMessage.Content))
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

	c.currentMessage = &types.Message{
		ID:      messageID,
		Role:    role,
		Content: "",
	}
	log.Printf("Compaction: TEXT_MESSAGE_START role=%s, messageId=%s", role, messageID)
}

func (c *MessageCompactor) handleTextMessageContent(event map[string]interface{}) {
	if c.currentMessage == nil {
		log.Printf("Compaction: WARNING - TEXT_MESSAGE_CONTENT without START")
		return
	}

	delta, _ := event["delta"].(string)
	c.currentMessage.Content += delta
	log.Printf("Compaction: TEXT_MESSAGE_CONTENT delta=%d chars, total=%d chars", len(delta), len(c.currentMessage.Content))
}

func (c *MessageCompactor) handleTextMessageEnd(event map[string]interface{}) {
	if c.currentMessage != nil {
		// User messages never have tool calls - flush immediately
		if c.currentMessage.Role == types.RoleUser {
			log.Printf("Compaction: TEXT_MESSAGE_END (user) id=%s, flushing immediately", c.currentMessage.ID)
			c.messages = append(c.messages, *c.currentMessage)
			c.currentMessage = nil
		} else {
			// Assistant messages might have tool calls - keep open
			log.Printf("Compaction: TEXT_MESSAGE_END id=%s, content=%d chars (keeping message open for tool calls)", c.currentMessage.ID, len(c.currentMessage.Content))
			// We'll flush when a new TEXT_MESSAGE_START arrives or at the end of compaction
		}
	} else {
		log.Printf("Compaction: WARNING - TEXT_MESSAGE_END without current message")
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
		if parentToolUseID != "" {
			log.Printf("Compaction: Started tool %s (%s) with parent %s", toolName, toolID[:8], parentToolUseID[:8])
		} else {
			log.Printf("Compaction: Started tool %s (%s)", toolName, toolID[:8])
		}
	} else {
		log.Printf("Compaction: WARNING - TOOL_CALL_START without toolCallId")
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
		log.Printf("Compaction: WARNING - TOOL_CALL_ARGS without toolCallId")
		return
	}

	if active, ok := c.activeToolCalls[toolID]; ok {
		active.Args += delta
		log.Printf("Compaction: Accumulated args for %s: %d chars", active.Name, len(active.Args))
	} else {
		log.Printf("Compaction: WARNING - TOOL_CALL_ARGS for unknown tool %s", toolID[:8])
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
		log.Printf("Compaction: WARNING - TOOL_CALL_END without toolCallId")
		return
	}

	active, ok := c.activeToolCalls[toolID]
	if !ok {
		log.Printf("Compaction: WARNING - TOOL_CALL_END for unknown tool %s", toolID[:8])
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

	log.Printf("Compaction: Completed tool %s (%s), args=%d chars, result=%d chars",
		tc.Name, tc.ID[:8], len(tc.Args), len(tc.Result))

	// Add to message
	// Check if we need to create a new message or add to current
	if c.currentMessage != nil && c.currentMessage.Role == types.RoleAssistant {
		// Add to current message
		c.currentMessage.ToolCalls = append(c.currentMessage.ToolCalls, tc)
	} else {
		// Create new message for this tool call
		c.messages = append(c.messages, types.Message{
			ID:        uuid.New().String(),
			Role:      types.RoleAssistant,
			ToolCalls: []types.ToolCall{tc},
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
				log.Printf("Compaction: Marking message %s as hidden", messageID[:8])
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

	log.Printf("Compaction: Received MESSAGES_SNAPSHOT with %d messages from runner", len(c.messages))
}

// CompactEvents is the main entry point for event compaction
func CompactEvents(events []map[string]interface{}) []types.Message {
	log.Printf("Compaction: Starting compaction of %d events", len(events))

	// Count event types to help debug
	eventTypeCounts := make(map[string]int)
	for _, event := range events {
		eventType, _ := event["type"].(string)
		eventTypeCounts[eventType]++
	}
	log.Printf("Compaction: Event type breakdown: %v", eventTypeCounts)

	compactor := NewMessageCompactor()

	for i, event := range events {
		eventType, _ := event["type"].(string)
		if i < 10 || i >= len(events)-10 {
			// Log first and last 10 events in detail
			log.Printf("Compaction: [%d/%d] Processing event type=%s", i+1, len(events), eventType)
		}
		compactor.HandleEvent(event)
	}

	messages := compactor.GetMessages()
	log.Printf("Compaction: Finished compaction - produced %d messages from %d events", len(messages), len(events))

	// Log summary of messages
	for i, msg := range messages {
		toolCallCount := len(msg.ToolCalls)
		log.Printf("Compaction:   Message[%d]: role=%s, id=%s, content=%d chars, toolCalls=%d",
			i, msg.Role, msg.ID, len(msg.Content), toolCallCount)
	}

	if len(messages) == 0 && len(events) > 0 {
		log.Printf("Compaction: WARNING - 0 messages produced! This indicates:")
		log.Printf("Compaction:   - No TEXT_MESSAGE_START events found, OR")
		log.Printf("Compaction:   - No MESSAGES_SNAPSHOT events found, OR")
		log.Printf("Compaction:   - Only lifecycle events (which are skipped)")
	}

	return messages
}
