/**
 * AG-UI Protocol Types
 * TypeScript types for AG-UI events and messages.
 * 
 * Reference: https://docs.ag-ui.com/concepts/events
 * Reference: https://docs.ag-ui.com/concepts/messages
 */

// AG-UI Event Types
export const AGUIEventType = {
  // Lifecycle events
  RUN_STARTED: 'RUN_STARTED',
  RUN_FINISHED: 'RUN_FINISHED',
  RUN_ERROR: 'RUN_ERROR',

  // Step events
  STEP_STARTED: 'STEP_STARTED',
  STEP_FINISHED: 'STEP_FINISHED',

  // Text message events (streaming)
  TEXT_MESSAGE_START: 'TEXT_MESSAGE_START',
  TEXT_MESSAGE_CONTENT: 'TEXT_MESSAGE_CONTENT',
  TEXT_MESSAGE_END: 'TEXT_MESSAGE_END',

  // Tool call events (streaming)
  TOOL_CALL_START: 'TOOL_CALL_START',
  TOOL_CALL_ARGS: 'TOOL_CALL_ARGS',
  TOOL_CALL_END: 'TOOL_CALL_END',

  // State management events
  STATE_SNAPSHOT: 'STATE_SNAPSHOT',
  STATE_DELTA: 'STATE_DELTA',

  // Message snapshot for restore/reconnect
  MESSAGES_SNAPSHOT: 'MESSAGES_SNAPSHOT',

  // Activity events
  ACTIVITY_SNAPSHOT: 'ACTIVITY_SNAPSHOT',
  ACTIVITY_DELTA: 'ACTIVITY_DELTA',

  // Raw event
  RAW: 'RAW',
  
  // Meta events (user feedback, annotations, etc.)
  META: 'META',
} as const

export type AGUIEventTypeValue = (typeof AGUIEventType)[keyof typeof AGUIEventType]

// AG-UI Message Roles
export const AGUIRole = {
  USER: 'user',
  ASSISTANT: 'assistant',
  SYSTEM: 'system',
  TOOL: 'tool',
  DEVELOPER: 'developer',
  ACTIVITY: 'activity',
} as const

export type AGUIRoleValue = (typeof AGUIRole)[keyof typeof AGUIRole]

// Base event structure
export type AGUIBaseEvent = {
  type: AGUIEventTypeValue
  threadId: string
  runId: string
  timestamp: string
  messageId?: string
  parentRunId?: string
}

// Run input/output types
export type AGUIRunAgentInput = {
  threadId?: string
  runId?: string
  parentRunId?: string
  messages?: AGUIMessage[]
  state?: Record<string, unknown>
  tools?: AGUIToolDefinition[]
  context?: Record<string, unknown>
}

export type AGUIRunAgentOutput = {
  threadId: string
  runId: string
  parentRunId?: string
  streamUrl?: string
}

// Message type
export type AGUIMessage = {
  id: string
  role: AGUIRoleValue
  content?: string
  toolCalls?: AGUIToolCall[]
  toolCallId?: string
  name?: string
  timestamp?: string
  metadata?: unknown
  parentToolUseId?: string  // For hierarchical tool calls (sub-agents)
  children?: AGUIMessage[]  // Nested tool calls under this tool
}

// Tool types
export type AGUIToolCall = {
  id: string
  name: string
  args: string
  type?: string
  parentToolUseId?: string  // For parent-child relationships (sub-agents)
  result?: string
  status?: 'pending' | 'running' | 'completed' | 'error'
  error?: string
  duration?: number
}

export type AGUIToolDefinition = {
  name: string
  description?: string
  parameters?: Record<string, unknown>
}

// Lifecycle events
export type AGUIRunStartedEvent = AGUIBaseEvent & {
  type: typeof AGUIEventType.RUN_STARTED
  input?: AGUIRunAgentInput
}

export type AGUIRunFinishedEvent = AGUIBaseEvent & {
  type: typeof AGUIEventType.RUN_FINISHED
  output?: unknown
}

export type AGUIRunErrorEvent = AGUIBaseEvent & {
  type: typeof AGUIEventType.RUN_ERROR
  error: string
  code?: string
  details?: string
}

// Step events
export type AGUIStepStartedEvent = AGUIBaseEvent & {
  type: typeof AGUIEventType.STEP_STARTED
  stepId: string
  stepName: string
}

export type AGUIStepFinishedEvent = AGUIBaseEvent & {
  type: typeof AGUIEventType.STEP_FINISHED
  stepId: string
  stepName: string
  duration?: number
}

// Text message events
export type AGUITextMessageStartEvent = AGUIBaseEvent & {
  type: typeof AGUIEventType.TEXT_MESSAGE_START
  role: AGUIRoleValue
}

export type AGUITextMessageContentEvent = AGUIBaseEvent & {
  type: typeof AGUIEventType.TEXT_MESSAGE_CONTENT
  delta: string
}

export type AGUITextMessageEndEvent = AGUIBaseEvent & {
  type: typeof AGUIEventType.TEXT_MESSAGE_END
}

// Tool call events
export type AGUIToolCallStartEvent = AGUIBaseEvent & {
  type: typeof AGUIEventType.TOOL_CALL_START
  toolCallId: string
  toolCallName: string
  parentMessageId?: string
  parentToolUseId?: string
}

export type AGUIToolCallArgsEvent = AGUIBaseEvent & {
  type: typeof AGUIEventType.TOOL_CALL_ARGS
  toolCallId: string
  delta: string
}

export type AGUIToolCallEndEvent = AGUIBaseEvent & {
  type: typeof AGUIEventType.TOOL_CALL_END
  toolCallId: string
  result?: string
  error?: string
  duration?: number
}

// State events
export type AGUIStateSnapshotEvent = AGUIBaseEvent & {
  type: typeof AGUIEventType.STATE_SNAPSHOT
  state: Record<string, unknown>
}

export type AGUIStateDeltaEvent = AGUIBaseEvent & {
  type: typeof AGUIEventType.STATE_DELTA
  delta: AGUIStatePatch[]
}

export type AGUIStatePatch = {
  op: 'add' | 'remove' | 'replace'
  path: string
  value?: unknown
}

// Message snapshot event
export type AGUIMessagesSnapshotEvent = AGUIBaseEvent & {
  type: typeof AGUIEventType.MESSAGES_SNAPSHOT
  messages: AGUIMessage[]
}

// Activity types
export type AGUIActivity = {
  id: string
  type: string
  title?: string
  status?: 'pending' | 'running' | 'completed' | 'error'
  progress?: number
  data?: Record<string, unknown>
}

export type AGUIActivitySnapshotEvent = AGUIBaseEvent & {
  type: typeof AGUIEventType.ACTIVITY_SNAPSHOT
  activities: AGUIActivity[]
}

export type AGUIActivityDeltaEvent = AGUIBaseEvent & {
  type: typeof AGUIEventType.ACTIVITY_DELTA
  delta: AGUIActivityPatch[]
}

export type AGUIActivityPatch = {
  op: 'add' | 'update' | 'remove'
  activity: AGUIActivity
}

// Raw event
export type AGUIRawEvent = AGUIBaseEvent & {
  type: typeof AGUIEventType.RAW
  data: unknown
}

// Meta event (user feedback, annotations, etc.)
export type AGUIMetaEvent = {
  type: typeof AGUIEventType.META
  metaType: string  // e.g., 'thumbs_up', 'thumbs_down'
  payload: Record<string, unknown>
  threadId: string
  ts?: number  // Unix timestamp in milliseconds
}

// Union of all event types
export type AGUIEvent =
  | AGUIRunStartedEvent
  | AGUIRunFinishedEvent
  | AGUIRunErrorEvent
  | AGUIStepStartedEvent
  | AGUIStepFinishedEvent
  | AGUITextMessageStartEvent
  | AGUITextMessageContentEvent
  | AGUITextMessageEndEvent
  | AGUIToolCallStartEvent
  | AGUIToolCallArgsEvent
  | AGUIToolCallEndEvent
  | AGUIStateSnapshotEvent
  | AGUIStateDeltaEvent
  | AGUIMessagesSnapshotEvent
  | AGUIActivitySnapshotEvent
  | AGUIActivityDeltaEvent
  | AGUIRawEvent
  | AGUIMetaEvent

// Run metadata type
export type AGUIRunMetadata = {
  threadId: string
  runId: string
  parentRunId?: string
  sessionName: string
  projectName: string
  startedAt: string
  finishedAt?: string
  status: 'running' | 'completed' | 'error'
  eventCount?: number
  restartCount?: number
}

// History response type
export type AGUIHistoryResponse = {
  threadId: string
  runId?: string
  messages: AGUIMessage[]
  runs: AGUIRunMetadata[]
}

// Runs response type
export type AGUIRunsResponse = {
  threadId: string
  runs: AGUIRunMetadata[]
}

// Pending tool call being streamed
export type PendingToolCall = {
  id: string
  name: string
  args: string
  parentToolUseId?: string
  timestamp?: string  // Timestamp from TOOL_CALL_START event
}

// Feedback type for messages
export type MessageFeedback = 'thumbs_up' | 'thumbs_down'

// Client state for AG-UI streaming
export type AGUIClientState = {
  threadId: string | null
  runId: string | null
  status: 'idle' | 'connecting' | 'connected' | 'error' | 'completed'
  messages: AGUIMessage[]
  state: Record<string, unknown>
  activities: AGUIActivity[]
  currentMessage: {
    id: string | null
    role: AGUIRoleValue | null
    content: string
    timestamp?: string  // Timestamp from TEXT_MESSAGE_START event
  } | null
  // DEPRECATED: Use pendingToolCalls instead for parallel tool call support
  currentToolCall: {
    id: string | null
    name: string | null
    args: string
    parentToolUseId?: string
  } | null
  // Track ALL in-progress tool calls (supports parallel tool execution)
  pendingToolCalls: Map<string, PendingToolCall>
  // Track child tools that finished before their parent
  pendingChildren: Map<string, AGUIMessage[]>
  error: string | null
  // Track feedback for messages (messageId -> feedback type)
  messageFeedback: Map<string, MessageFeedback>
}

// Type guard functions
export function isRunStartedEvent(event: AGUIEvent): event is AGUIRunStartedEvent {
  return event.type === AGUIEventType.RUN_STARTED
}

export function isRunFinishedEvent(event: AGUIEvent): event is AGUIRunFinishedEvent {
  return event.type === AGUIEventType.RUN_FINISHED
}

export function isRunErrorEvent(event: AGUIEvent): event is AGUIRunErrorEvent {
  return event.type === AGUIEventType.RUN_ERROR
}

export function isTextMessageStartEvent(event: AGUIEvent): event is AGUITextMessageStartEvent {
  return event.type === AGUIEventType.TEXT_MESSAGE_START
}

export function isTextMessageContentEvent(event: AGUIEvent): event is AGUITextMessageContentEvent {
  return event.type === AGUIEventType.TEXT_MESSAGE_CONTENT
}

export function isTextMessageEndEvent(event: AGUIEvent): event is AGUITextMessageEndEvent {
  return event.type === AGUIEventType.TEXT_MESSAGE_END
}

export function isToolCallStartEvent(event: AGUIEvent): event is AGUIToolCallStartEvent {
  return event.type === AGUIEventType.TOOL_CALL_START
}

export function isToolCallEndEvent(event: AGUIEvent): event is AGUIToolCallEndEvent {
  return event.type === AGUIEventType.TOOL_CALL_END
}

export function isStateSnapshotEvent(event: AGUIEvent): event is AGUIStateSnapshotEvent {
  return event.type === AGUIEventType.STATE_SNAPSHOT
}

export function isMessagesSnapshotEvent(event: AGUIEvent): event is AGUIMessagesSnapshotEvent {
  return event.type === AGUIEventType.MESSAGES_SNAPSHOT
}

export function isActivitySnapshotEvent(event: AGUIEvent): event is AGUIActivitySnapshotEvent {
  return event.type === AGUIEventType.ACTIVITY_SNAPSHOT
}

