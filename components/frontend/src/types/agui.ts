/**
 * AG-UI Protocol Types
 *
 * Re-exports canonical types from @ag-ui/client (@ag-ui/core) and defines
 * platform-specific extensions for hierarchical tool calls, message metadata,
 * and streaming client state.
 *
 * Reference: https://docs.ag-ui.com/concepts/events
 * Reference: https://docs.ag-ui.com/concepts/messages
 */

// ── Core AG-UI types (re-exported from @ag-ui/client) ──

export {
  EventType,
  type BaseEvent,
  type RunAgentInput,
  type AGUIEvent,
  type Message,
  type Role,
  type ToolCall,
  type FunctionCall,
  type Tool,
  type Context,
  type State,
  type RunStartedEvent,
  type RunFinishedEvent,
  type RunErrorEvent,
  type TextMessageStartEvent,
  type TextMessageContentEvent,
  type TextMessageEndEvent,
  type ToolCallStartEvent,
  type ToolCallArgsEvent,
  type ToolCallEndEvent,
  type ToolCallResultEvent,
  type StateSnapshotEvent,
  type StateDeltaEvent,
  type MessagesSnapshotEvent,
  type StepStartedEvent,
  type StepFinishedEvent,
  type ReasoningStartEvent,
  type ReasoningMessageStartEvent,
  type ReasoningMessageContentEvent,
  type ReasoningMessageEndEvent,
  type ReasoningEndEvent,
} from '@ag-ui/client'

import { EventType } from '@ag-ui/client'
import type {
  ToolCall,
  Message,
  RunStartedEvent,
  RunFinishedEvent,
  RunErrorEvent,
  TextMessageStartEvent,
  TextMessageContentEvent,
  TextMessageEndEvent,
  ToolCallStartEvent,
  ToolCallEndEvent,
  StateSnapshotEvent,
  MessagesSnapshotEvent,
} from '@ag-ui/client'

// ── Platform Extension: PlatformToolCall ──
// Extends core ToolCall with platform-specific tracking fields
// for hierarchical tool calls (sub-agents), result caching, and timing.
export type PlatformToolCall = ToolCall & {
  parentToolUseId?: string
  result?: string
  status?: 'pending' | 'running' | 'completed' | 'error'
  error?: string
  duration?: number
}

// ── Platform Extension: PlatformMessage ──
// Extends core Message union with platform-specific fields.
// Because Message is a discriminated union (A | B | C), the intersection
// distributes: (A & Ext) | (B & Ext) | (C & Ext), preserving discrimination.
export type PlatformMessage = Message & {
  timestamp?: string
  metadata?: unknown
  name?: string  // Tool name (not on core ToolMessage, but platform sends it)
  toolCalls?: PlatformToolCall[]
  toolCallId?: string  // Present on tool-role messages
  parentToolUseId?: string
  children?: PlatformMessage[]
}

// ── Wire event types for platform-specific fields ──
// The backend sends extra fields not in the core schema. Since AG-UI schemas
// use "passthrough", these survive JSON parsing but need explicit types.

/** ToolCallStartEvent with backend's extra snake_case parent field */
export type WireToolCallStartEvent = ToolCallStartEvent & {
  parent_tool_call_id?: string
}

/** Platform raw event — extends core RawEvent with optional data field.
 *  The backend may send payload in either `event` or `data`. */
export type PlatformRawEvent = {
  type: typeof EventType.RAW
  timestamp?: number
  event?: Record<string, unknown>
  data?: Record<string, unknown>
  source?: string
}

// ── Platform Activity types ──
// The core ActivitySnapshotEvent/ActivityDeltaEvent are per-message, not
// array-based. The platform uses an array-based model for UI rendering.

export type PlatformActivity = {
  id: string
  type: string
  title?: string
  status?: 'pending' | 'running' | 'completed' | 'error'
  progress?: number
  data?: Record<string, unknown>
}

export type PlatformActivityPatch = {
  op: 'add' | 'update' | 'remove'
  activity: PlatformActivity
}

/** Platform's array-based ACTIVITY_SNAPSHOT (differs from core's per-message model) */
export type PlatformActivitySnapshotEvent = {
  type: typeof EventType.ACTIVITY_SNAPSHOT
  timestamp?: number
  activities?: PlatformActivity[]
}

/** Platform's array-based ACTIVITY_DELTA (differs from core's per-message model) */
export type PlatformActivityDeltaEvent = {
  type: typeof EventType.ACTIVITY_DELTA
  timestamp?: number
  delta: PlatformActivityPatch[]
}

// ── Platform-specific types (no core equivalent) ──

// Meta event (user feedback, annotations, etc.)
export type AGUIMetaEvent = {
  type: 'META'
  metaType: string
  payload: Record<string, unknown>
  threadId: string
  ts?: number
}

// Union of all events the platform handles (core + META)
export type PlatformEvent = import('@ag-ui/client').AGUIEvent | AGUIMetaEvent

// Pending tool call during streaming (flat format for accumulation)
export type PendingToolCall = {
  id: string
  name: string
  args: string
  parentToolUseId?: string
  parentMessageId?: string
  timestamp?: string
}

// Feedback type for messages
export type MessageFeedback = 'thumbs_up' | 'thumbs_down'

// Client state for AG-UI streaming
export type AGUIClientState = {
  threadId: string | null
  runId: string | null
  status: 'idle' | 'connecting' | 'connected' | 'error' | 'completed'
  messages: PlatformMessage[]
  state: Record<string, unknown>
  activities: PlatformActivity[]
  currentMessage: {
    id: string | null
    role: string | null
    content: string
    timestamp?: string
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
  pendingChildren: Map<string, PlatformMessage[]>
  error: string | null
  // Track feedback for messages (messageId -> feedback type)
  messageFeedback: Map<string, MessageFeedback>
  // Buffer for reasoning content during streaming
  currentReasoning: {
    id: string | null
    content: string
    timestamp?: string
  } | null
  // Buffer for thinking content during streaming (THINKING_* events)
  currentThinking: {
    id: string | null
    content: string
    timestamp?: string
  } | null
}

// ── Type Guards ──
// Narrow parsed SSE events to specific core event types.

export function isRunStartedEvent(event: { type: string }): event is RunStartedEvent {
  return event.type === EventType.RUN_STARTED
}

export function isRunFinishedEvent(event: { type: string }): event is RunFinishedEvent {
  return event.type === EventType.RUN_FINISHED
}

export function isRunErrorEvent(event: { type: string }): event is RunErrorEvent {
  return event.type === EventType.RUN_ERROR
}

export function isTextMessageStartEvent(event: { type: string }): event is TextMessageStartEvent {
  return event.type === EventType.TEXT_MESSAGE_START
}

export function isTextMessageContentEvent(event: { type: string }): event is TextMessageContentEvent {
  return event.type === EventType.TEXT_MESSAGE_CONTENT
}

export function isTextMessageEndEvent(event: { type: string }): event is TextMessageEndEvent {
  return event.type === EventType.TEXT_MESSAGE_END
}

export function isToolCallStartEvent(event: { type: string }): event is ToolCallStartEvent {
  return event.type === EventType.TOOL_CALL_START
}

export function isToolCallEndEvent(event: { type: string }): event is ToolCallEndEvent {
  return event.type === EventType.TOOL_CALL_END
}

export function isStateSnapshotEvent(event: { type: string }): event is StateSnapshotEvent {
  return event.type === EventType.STATE_SNAPSHOT
}

export function isMessagesSnapshotEvent(event: { type: string }): event is MessagesSnapshotEvent {
  return event.type === EventType.MESSAGES_SNAPSHOT
}

export function isActivitySnapshotEvent(event: { type: string }): event is PlatformActivitySnapshotEvent {
  return event.type === EventType.ACTIVITY_SNAPSHOT
}
