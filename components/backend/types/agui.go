// Package types defines AG-UI protocol types for event streaming.
// Reference: https://docs.ag-ui.com/concepts/events
package types

import "time"

// Timestamp format constants for AG-UI events and metadata.
// These ensure consistent timestamp formatting across the codebase.
const (
	// AGUITimestampFormat is used for event timestamps that require nanosecond precision.
	// This preserves event ordering when multiple events occur in rapid succession.
	// Format: "2006-01-02T15:04:05.999999999Z07:00" (RFC3339 with nanoseconds)
	// Used in: BaseEvent.Timestamp, streamed events
	AGUITimestampFormat = time.RFC3339Nano

	// AGUIMetadataTimestampFormat is used for run/session metadata timestamps.
	// This is sufficient for human-readable timestamps where nanosecond precision isn't needed.
	// Format: "2006-01-02T15:04:05Z07:00" (RFC3339)
	// Used in: AGUIRunMetadata.StartedAt, AGUIRunMetadata.FinishedAt
	AGUIMetadataTimestampFormat = time.RFC3339
)

// AG-UI Event Types as defined in the protocol specification
// See: https://docs.ag-ui.com/concepts/events
const (
	// Lifecycle events
	EventTypeRunStarted  = "RUN_STARTED"
	EventTypeRunFinished = "RUN_FINISHED"
	EventTypeRunError    = "RUN_ERROR"

	// Step events
	EventTypeStepStarted  = "STEP_STARTED"
	EventTypeStepFinished = "STEP_FINISHED"

	// Text message events (streaming)
	EventTypeTextMessageStart   = "TEXT_MESSAGE_START"
	EventTypeTextMessageContent = "TEXT_MESSAGE_CONTENT"
	EventTypeTextMessageEnd     = "TEXT_MESSAGE_END"

	// Tool call events (streaming)
	EventTypeToolCallStart = "TOOL_CALL_START"
	EventTypeToolCallArgs  = "TOOL_CALL_ARGS"
	EventTypeToolCallEnd   = "TOOL_CALL_END"

	// State management events
	EventTypeStateSnapshot = "STATE_SNAPSHOT"
	EventTypStateDelta     = "STATE_DELTA"

	// Message snapshot for restore/reconnect
	EventTypeMessagesSnapshot = "MESSAGES_SNAPSHOT"

	// Activity events (frontend-only durable UI)
	EventTypeActivitySnapshot = "ACTIVITY_SNAPSHOT"
	EventTypeActivityDelta    = "ACTIVITY_DELTA"

	// Raw event for pass-through
	EventTypeRaw = "RAW"

	// META event for user feedback (thumbs up/down)
	// See: https://docs.ag-ui.com/drafts/meta-events
	EventTypeMeta = "META"
)

// AG-UI Message Roles
// See: https://docs.ag-ui.com/concepts/messages
const (
	RoleUser      = "user"
	RoleAssistant = "assistant"
	RoleSystem    = "system"
	RoleTool      = "tool"
	RoleDeveloper = "developer"
	RoleActivity  = "activity"
)

// BaseEvent is the common structure for all AG-UI events
// See: https://docs.ag-ui.com/concepts/events#baseeventproperties
type BaseEvent struct {
	Type      string `json:"type"`
	ThreadID  string `json:"threadId"`
	RunID     string `json:"runId"`
	Timestamp string `json:"timestamp"` // Format: AGUITimestampFormat (RFC3339Nano)
	// Optional fields
	MessageID   string `json:"messageId,omitempty"`
	ParentRunID string `json:"parentRunId,omitempty"`
}

// RunAgentInput is the input format for starting an AG-UI run
// See: https://docs.ag-ui.com/quickstart/introduction
type RunAgentInput struct {
	ThreadID    string                 `json:"threadId,omitempty"`
	RunID       string                 `json:"runId,omitempty"`
	ParentRunID string                 `json:"parentRunId,omitempty"`
	Messages    []Message              `json:"messages,omitempty"`
	State       map[string]interface{} `json:"state,omitempty"`
	Tools       []ToolDefinition       `json:"tools,omitempty"`
	Context     map[string]interface{} `json:"context,omitempty"`
}

// RunAgentOutput is the response after starting a run
type RunAgentOutput struct {
	ThreadID    string `json:"threadId"`
	RunID       string `json:"runId"`
	ParentRunID string `json:"parentRunId,omitempty"`
	StreamURL   string `json:"streamUrl,omitempty"`
}

// Message represents an AG-UI message in the conversation
// See: https://docs.ag-ui.com/concepts/messages
type Message struct {
	ID         string      `json:"id"`
	Role       string      `json:"role"`
	Content    string      `json:"content,omitempty"`
	ToolCalls  []ToolCall  `json:"toolCalls,omitempty"`
	ToolCallID string      `json:"toolCallId,omitempty"`
	Name       string      `json:"name,omitempty"`
	Timestamp  string      `json:"timestamp,omitempty"`
	Metadata   interface{} `json:"metadata,omitempty"`
}

// ToolCall represents a tool call made by the assistant
type ToolCall struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Args            string `json:"args"`
	Type            string `json:"type,omitempty"`            // "function"
	ParentToolUseID string `json:"parentToolUseId,omitempty"` // For hierarchical nesting
	Result          string `json:"result,omitempty"`
	Status          string `json:"status,omitempty"` // "pending", "running", "completed", "error"
	Error           string `json:"error,omitempty"`
	Duration        int64  `json:"duration,omitempty"` // milliseconds
}

// ToolDefinition describes an available tool
type ToolDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

// RunStartedEvent is emitted when a run begins
type RunStartedEvent struct {
	BaseEvent
	Input *RunAgentInput `json:"input,omitempty"`
}

// RunFinishedEvent is emitted when a run completes successfully
type RunFinishedEvent struct {
	BaseEvent
	Output interface{} `json:"output,omitempty"`
}

// RunErrorEvent is emitted when a run fails
type RunErrorEvent struct {
	BaseEvent
	Error   string `json:"error"`
	Code    string `json:"code,omitempty"`
	Details string `json:"details,omitempty"`
}

// StepStartedEvent marks the beginning of a processing step
type StepStartedEvent struct {
	BaseEvent
	StepID   string `json:"stepId"`
	StepName string `json:"stepName"`
}

// StepFinishedEvent marks the completion of a processing step
type StepFinishedEvent struct {
	BaseEvent
	StepID   string `json:"stepId"`
	StepName string `json:"stepName"`
	Duration int64  `json:"duration,omitempty"` // milliseconds
}

// TextMessageStartEvent begins a streaming text message
type TextMessageStartEvent struct {
	BaseEvent
	Role string `json:"role"`
}

// TextMessageContentEvent contains a chunk of text content
type TextMessageContentEvent struct {
	BaseEvent
	Delta string `json:"delta"`
}

// TextMessageEndEvent marks the end of a streaming text message
type TextMessageEndEvent struct {
	BaseEvent
}

// ToolCallStartEvent begins a streaming tool call
type ToolCallStartEvent struct {
	BaseEvent
	ToolCallID      string `json:"toolCallId"`
	ToolCallName    string `json:"toolCallName"`
	ParentMessageID string `json:"parentMessageId,omitempty"`
	ParentToolUseID string `json:"parentToolUseId,omitempty"`
}

// ToolCallArgsEvent contains a chunk of tool call arguments
type ToolCallArgsEvent struct {
	BaseEvent
	ToolCallID string `json:"toolCallId"`
	Delta      string `json:"delta"`
}

// ToolCallEndEvent marks the end of a streaming tool call
type ToolCallEndEvent struct {
	BaseEvent
	ToolCallID string `json:"toolCallId"`
	Result     string `json:"result,omitempty"`
	Error      string `json:"error,omitempty"`
	Duration   int64  `json:"duration,omitempty"` // milliseconds
}

// StateSnapshotEvent provides complete state for hydration
type StateSnapshotEvent struct {
	BaseEvent
	State map[string]interface{} `json:"state"`
}

// StateDeltaEvent provides incremental state updates
type StateDeltaEvent struct {
	BaseEvent
	Delta []StatePatch `json:"delta"`
}

// StatePatch represents a JSON Patch operation for state updates
type StatePatch struct {
	Op    string      `json:"op"`   // "add", "remove", "replace"
	Path  string      `json:"path"` // JSON Pointer
	Value interface{} `json:"value,omitempty"`
}

// MessagesSnapshotEvent provides complete message history for hydration
type MessagesSnapshotEvent struct {
	BaseEvent
	Messages []Message `json:"messages"`
}

// ActivitySnapshotEvent provides complete activity UI state
type ActivitySnapshotEvent struct {
	BaseEvent
	Activities []Activity `json:"activities"`
}

// ActivityDeltaEvent provides incremental activity updates
type ActivityDeltaEvent struct {
	BaseEvent
	Delta []ActivityPatch `json:"delta"`
}

// Activity represents a durable frontend UI element
type Activity struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"type"`
	Title    string                 `json:"title,omitempty"`
	Status   string                 `json:"status,omitempty"` // "pending", "running", "completed", "error"
	Progress float64                `json:"progress,omitempty"`
	Data     map[string]interface{} `json:"data,omitempty"`
}

// ActivityPatch represents an update to an activity
type ActivityPatch struct {
	Op       string   `json:"op"` // "add", "update", "remove"
	Activity Activity `json:"activity"`
}

// RawEvent allows pass-through of arbitrary data
type RawEvent struct {
	BaseEvent
	Data interface{} `json:"data"`
}

// MetaEvent represents AG-UI META events for user feedback
// See: https://docs.ag-ui.com/drafts/meta-events#user-feedback
type MetaEvent struct {
	BaseEvent
	MetaType string                 `json:"metaType"` // "thumbs_up" or "thumbs_down"
	Payload  map[string]interface{} `json:"payload"`
}

// FeedbackPayload contains the payload for feedback META events
type FeedbackPayload struct {
	MessageID string `json:"messageId,omitempty"` // ID of the message being rated
	UserID    string `json:"userId"`              // User providing feedback
	Reason    string `json:"reason,omitempty"`    // Reason for negative feedback
	Comment   string `json:"comment,omitempty"`   // Additional user comment
	// Extended fields for Langfuse context
	ProjectName       string                   `json:"projectName,omitempty"`
	SessionName       string                   `json:"sessionName,omitempty"`
	Workflow          string                   `json:"workflow,omitempty"`
	Context           string                   `json:"context,omitempty"`
	IncludeTranscript bool                     `json:"includeTranscript,omitempty"`
	Transcript        []FeedbackTranscriptItem `json:"transcript,omitempty"`
}

// FeedbackTranscriptItem represents a message in the feedback transcript
type FeedbackTranscriptItem struct {
	Role      string `json:"role"`
	Content   string `json:"content"`
	Timestamp string `json:"timestamp,omitempty"`
}

// NewBaseEvent creates a new BaseEvent with current timestamp
func NewBaseEvent(eventType, threadID, runID string) BaseEvent {
	return BaseEvent{
		Type:      eventType,
		ThreadID:  threadID,
		RunID:     runID,
		Timestamp: time.Now().UTC().Format(AGUITimestampFormat),
	}
}

// WithMessageID adds a message ID to the event
func (e BaseEvent) WithMessageID(messageID string) BaseEvent {
	e.MessageID = messageID
	return e
}

// WithParentRunID adds a parent run ID to the event
func (e BaseEvent) WithParentRunID(parentRunID string) BaseEvent {
	e.ParentRunID = parentRunID
	return e
}

// AGUIEventLog represents the persisted event log structure
type AGUIEventLog struct {
	ThreadID    string      `json:"threadId"`
	RunID       string      `json:"runId"`
	ParentRunID string      `json:"parentRunId,omitempty"`
	Events      []BaseEvent `json:"events"`
	CreatedAt   string      `json:"createdAt"`
	UpdatedAt   string      `json:"updatedAt"`
}

// AGUIRunMetadata contains metadata about a run for indexing
type AGUIRunMetadata struct {
	ThreadID     string `json:"threadId"`
	RunID        string `json:"runId"`
	ParentRunID  string `json:"parentRunId,omitempty"`
	SessionName  string `json:"sessionName"`
	ProjectName  string `json:"projectName"`
	StartedAt    string `json:"startedAt"`
	FinishedAt   string `json:"finishedAt,omitempty"`
	Status       string `json:"status"` // "running", "completed", "error"
	EventCount   int    `json:"eventCount"`
	RestartCount int    `json:"restartCount,omitempty"`
}
