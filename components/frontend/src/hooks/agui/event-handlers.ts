/**
 * AG-UI Event Handlers
 *
 * Pure functions that transform AGUIClientState in response to each event type.
 * Each handler takes (prev, event, callbacks) and returns the new state.
 */

import {
  EventType,
  isRunStartedEvent,
  isRunFinishedEvent,
  isRunErrorEvent,
  isTextMessageStartEvent,
  isTextMessageContentEvent,
  isTextMessageEndEvent,
  isToolCallStartEvent,
  isToolCallEndEvent,
  isStateSnapshotEvent,
  isMessagesSnapshotEvent,
  isActivitySnapshotEvent,
} from '@/types/agui'
import type {
  AGUIClientState,
  PlatformEvent,
  PlatformMessage,
  PlatformToolCall,
  PlatformRawEvent,
  AGUIMetaEvent,
  PlatformActivitySnapshotEvent,
  PlatformActivityDeltaEvent,
  WireToolCallStartEvent,
  RunStartedEvent,
  RunFinishedEvent,
  RunErrorEvent,
  TextMessageStartEvent,
  TextMessageContentEvent,
  TextMessageEndEvent,
  ToolCallArgsEvent,
  ToolCallEndEvent,
  ToolCallResultEvent,
  StateDeltaEvent,
  MessagesSnapshotEvent,
  StepStartedEvent,
  ReasoningMessageStartEvent,
  ReasoningMessageContentEvent,
  ReasoningMessageEndEvent,
} from '@/types/agui'
import { normalizeSnapshotMessages } from './normalize-snapshot'

/** Callbacks that event handlers may invoke for side effects */
export type EventHandlerCallbacks = {
  onMessage?: (message: PlatformMessage) => void
  onError?: (error: string) => void
  onTraceId?: (traceId: string) => void
  setIsRunActive: (active: boolean) => void
  currentRunIdRef: { current: string | null }
  hiddenMessageIdsRef: { current: Set<string> }
}

/**
 * Process a single AG-UI event and return the updated state.
 *
 * This is a pure state transition function (aside from the callbacks
 * which handle side effects like marking runs active).
 */
export function processAGUIEvent(
  prev: AGUIClientState,
  event: PlatformEvent,
  callbacks: EventHandlerCallbacks,
): AGUIClientState {
  const newState = { ...prev }

  if (isRunStartedEvent(event)) {
    return handleRunStarted(newState, event, callbacks)
  }

  if (isRunFinishedEvent(event)) {
    return handleRunFinished(newState, event, callbacks)
  }

  if (isRunErrorEvent(event)) {
    return handleRunError(newState, event, callbacks)
  }

  if (isTextMessageStartEvent(event)) {
    return handleTextMessageStart(newState, event)
  }

  if (isTextMessageContentEvent(event)) {
    return handleTextMessageContent(newState, event)
  }

  if (isTextMessageEndEvent(event)) {
    return handleTextMessageEnd(newState, event, callbacks)
  }

  if (isToolCallStartEvent(event)) {
    return handleToolCallStart(newState, event as WireToolCallStartEvent)
  }

  if (event.type === EventType.TOOL_CALL_ARGS) {
    return handleToolCallArgs(newState, event as ToolCallArgsEvent)
  }

  if (isToolCallEndEvent(event)) {
    return handleToolCallEnd(newState, event, callbacks)
  }

  if (event.type === EventType.TOOL_CALL_RESULT) {
    return handleToolCallResult(newState, event as ToolCallResultEvent)
  }

  if (isStateSnapshotEvent(event)) {
    newState.state = event.snapshot as Record<string, unknown>
    return newState
  }

  if (event.type === EventType.STATE_DELTA) {
    return handleStateDelta(newState, event as StateDeltaEvent)
  }

  if (isMessagesSnapshotEvent(event)) {
    return handleMessagesSnapshot(newState, event, callbacks)
  }

  if (isActivitySnapshotEvent(event)) {
    return handleActivitySnapshot(newState, event as PlatformActivitySnapshotEvent)
  }

  if (event.type === EventType.ACTIVITY_DELTA) {
    return handleActivityDelta(newState, event as unknown as PlatformActivityDeltaEvent)
  }

  if (event.type === EventType.STEP_STARTED) {
    return handleStepStarted(newState, event as StepStartedEvent)
  }

  if (event.type === EventType.STEP_FINISHED) {
    return handleStepFinished(newState)
  }

  // ── Reasoning events ──
  if (event.type === EventType.REASONING_START) {
    // Lifecycle bookend -- no-op (REASONING_MESSAGE_START does the real work)
    return newState
  }

  if (event.type === EventType.REASONING_MESSAGE_START) {
    return handleReasoningMessageStart(newState, event as ReasoningMessageStartEvent)
  }

  if (event.type === EventType.REASONING_MESSAGE_CONTENT) {
    return handleReasoningMessageContent(newState, event as ReasoningMessageContentEvent)
  }

  if (event.type === EventType.REASONING_MESSAGE_END) {
    return handleReasoningMessageEnd(newState, event, callbacks)
  }

  if (event.type === EventType.REASONING_END) {
    // Lifecycle bookend -- no-op
    return newState
  }

  if (event.type === EventType.RAW) {
    return handleRawEvent(newState, event as PlatformRawEvent, callbacks)
  }

  if (event.type === 'META') {
    return handleMetaEvent(newState, event as AGUIMetaEvent)
  }

  // ── Thinking events (custom, not in AG-UI EventType enum) ──
  if (event.type === 'THINKING_START') {
    return newState
  }

  if (event.type === 'THINKING_TEXT_MESSAGE_START') {
    return handleThinkingMessageStart(newState)
  }

  if (event.type === 'THINKING_TEXT_MESSAGE_CONTENT') {
    return handleThinkingMessageContent(newState, event as { type: string; delta: string })
  }

  if (event.type === 'THINKING_TEXT_MESSAGE_END') {
    return handleThinkingMessageEnd(newState, event, callbacks)
  }

  if (event.type === 'THINKING_END') {
    return newState
  }

  return newState
}

// ── Individual event handlers ──

function handleRunStarted(
  state: AGUIClientState,
  event: RunStartedEvent,
  callbacks: EventHandlerCallbacks,
): AGUIClientState {
  state.threadId = event.threadId
  state.runId = event.runId
  state.status = 'connected'
  state.error = null
  callbacks.currentRunIdRef.current = event.runId
  callbacks.setIsRunActive(true)
  return state
}

function handleRunFinished(
  state: AGUIClientState,
  event: RunFinishedEvent,
  callbacks: EventHandlerCallbacks,
): AGUIClientState {
  state.status = 'completed'

  if (callbacks.currentRunIdRef.current === event.runId) {
    callbacks.setIsRunActive(false)
    callbacks.currentRunIdRef.current = null
  }

  // Flush any pending message
  if (state.currentMessage?.content) {
    const msg = {
      id: state.currentMessage.id || crypto.randomUUID(),
      role: 'assistant' as const,
      content: state.currentMessage.content,
      timestamp: String(event.timestamp ?? ''),
    } as PlatformMessage
    state.messages = [...state.messages, msg]
    callbacks.onMessage?.(msg)
  }
  state.currentMessage = null

  // Flush any pending reasoning
  if (state.currentReasoning?.content) {
    const msg = {
      id: state.currentReasoning.id || crypto.randomUUID(),
      role: 'assistant' as const,
      content: state.currentReasoning.content,
      metadata: { type: 'reasoning_block' },
      timestamp: String(event.timestamp ?? ''),
    } as PlatformMessage
    state.messages = [...state.messages, msg]
    callbacks.onMessage?.(msg)
  }
  state.currentReasoning = null

  // Flush any pending thinking
  if (state.currentThinking?.content) {
    const thinkingText = state.currentThinking.content
    const msg = {
      id: state.currentThinking.id || crypto.randomUUID(),
      role: 'assistant' as const,
      content: thinkingText,
      metadata: {
        type: 'thinking_block',
        thinking: thinkingText,
        signature: '',
      },
      timestamp: String(event.timestamp ?? ''),
    } as PlatformMessage
    state.messages = [...state.messages, msg]
    callbacks.onMessage?.(msg)
  }
  state.currentThinking = null

  return state
}

function handleRunError(
  state: AGUIClientState,
  event: RunErrorEvent,
  callbacks: EventHandlerCallbacks,
): AGUIClientState {
  state.status = 'error'
  state.error = event.message
  callbacks.onError?.(event.message)
  callbacks.setIsRunActive(false)
  callbacks.currentRunIdRef.current = null
  return state
}

function handleTextMessageStart(
  state: AGUIClientState,
  event: TextMessageStartEvent,
): AGUIClientState {
  state.currentMessage = {
    id: event.messageId || null,
    role: event.role,
    content: '',
    timestamp: String(event.timestamp ?? ''),
  }
  return state
}

function handleTextMessageContent(
  state: AGUIClientState,
  event: TextMessageContentEvent,
): AGUIClientState {
  if (state.currentMessage) {
    state.currentMessage = {
      ...state.currentMessage,
      content: (state.currentMessage.content || '') + event.delta,
    }
  }
  return state
}

function handleTextMessageEnd(
  state: AGUIClientState,
  event: TextMessageEndEvent,
  callbacks: EventHandlerCallbacks,
): AGUIClientState {
  if (state.currentMessage?.content) {
    const messageId = state.currentMessage.id || crypto.randomUUID()

    // Skip hidden messages (auto-sent initial/workflow prompts)
    if (callbacks.hiddenMessageIdsRef.current.has(messageId)) {
      state.currentMessage = null
      return state
    }

    // Check if this message already exists (e.g., from MESSAGES_SNAPSHOT)
    const existingIndex = state.messages.findIndex(m => m.id === messageId)

    if (existingIndex >= 0) {
      const existingMsg = state.messages[existingIndex]
      if (existingMsg.content !== state.currentMessage.content) {
        const updatedMessages = [...state.messages]
        updatedMessages[existingIndex] = {
          ...existingMsg,
          content: state.currentMessage.content,
        } as PlatformMessage
        state.messages = updatedMessages
      }
    } else {
      const msg = {
        id: messageId,
        role: state.currentMessage.role || 'assistant',
        content: state.currentMessage.content,
        timestamp: String(event.timestamp ?? ''),
      } as PlatformMessage
      state.messages = [...state.messages, msg]
      callbacks.onMessage?.(msg)
    }
  }
  state.currentMessage = null
  return state
}

function handleToolCallStart(
  state: AGUIClientState,
  event: WireToolCallStartEvent,
): AGUIClientState {
  const parentToolId = event.parent_tool_call_id
  const parentMessageId = event.parentMessageId

  // Determine effective parent tool ID for hierarchy.
  let effectiveParentToolId = parentToolId
  if (!effectiveParentToolId && parentMessageId) {
    if (state.pendingToolCalls.has(parentMessageId)) {
      effectiveParentToolId = parentMessageId
    } else {
      for (let i = state.messages.length - 1; i >= 0; i--) {
        if (state.messages[i].toolCalls?.some(tc => tc.id === parentMessageId)) {
          effectiveParentToolId = parentMessageId
          break
        }
      }
    }
  }

  // Store in pendingToolCalls Map to support parallel tool calls
  const updatedPending = new Map(state.pendingToolCalls)
  updatedPending.set(event.toolCallId, {
    id: event.toolCallId,
    name: event.toolCallName || 'unknown_tool',
    args: '',
    parentToolUseId: effectiveParentToolId,
    parentMessageId: parentMessageId,
    timestamp: String(event.timestamp ?? ''),
  })
  state.pendingToolCalls = updatedPending

  // Also update currentToolCall for backward compat (UI rendering)
  state.currentToolCall = {
    id: event.toolCallId,
    name: event.toolCallName,
    args: '',
    parentToolUseId: effectiveParentToolId,
  }
  return state
}

function handleToolCallArgs(
  state: AGUIClientState,
  event: ToolCallArgsEvent,
): AGUIClientState {
  const toolCallId = event.toolCallId
  const existing = state.pendingToolCalls.get(toolCallId)

  if (existing) {
    const updatedPending = new Map(state.pendingToolCalls)
    updatedPending.set(toolCallId, {
      ...existing,
      args: (existing.args || '') + event.delta,
    })
    state.pendingToolCalls = updatedPending
  }

  // Also update currentToolCall for backward compat (if it's the same tool)
  if (state.currentToolCall?.id === toolCallId) {
    state.currentToolCall = {
      ...state.currentToolCall,
      args: (state.currentToolCall.args || '') + event.delta,
    }
  }
  return state
}

function handleToolCallEnd(
  state: AGUIClientState,
  event: ToolCallEndEvent,
  callbacks: EventHandlerCallbacks,
): AGUIClientState {
  const toolCallId = event.toolCallId || state.currentToolCall?.id || crypto.randomUUID()

  // Get tool info from pendingToolCalls Map (supports parallel tool calls)
  const pendingTool = state.pendingToolCalls.get(toolCallId)
  const toolCallName = pendingTool?.name || state.currentToolCall?.name || 'unknown_tool'
  const toolCallArgs = pendingTool?.args || state.currentToolCall?.args || ''
  const parentToolUseId = pendingTool?.parentToolUseId || state.currentToolCall?.parentToolUseId
  const parentMessageId = pendingTool?.parentMessageId

  // Defense in depth: Check if this tool already exists
  const toolAlreadyExists = state.messages.some(msg =>
    msg.toolCalls?.some(tc => tc.id === toolCallId)
  )

  if (toolAlreadyExists) {
    const updatedPendingTools = new Map(state.pendingToolCalls)
    updatedPendingTools.delete(toolCallId)
    state.pendingToolCalls = updatedPendingTools
    if (state.currentToolCall?.id === toolCallId) {
      state.currentToolCall = null
    }
    return state
  }

  // Create completed tool call using @ag-ui/core ToolCall format
  const completedToolCall: PlatformToolCall = {
    id: toolCallId,
    type: 'function',
    function: {
      name: toolCallName,
      arguments: toolCallArgs,
    },
    result: undefined as string | undefined,
    status: 'completed' as const,
    parentToolUseId: parentToolUseId,
  }

  const messages = [...state.messages]

  // Remove from pendingToolCalls Map
  const updatedPendingTools = new Map(state.pendingToolCalls)
  updatedPendingTools.delete(toolCallId)
  state.pendingToolCalls = updatedPendingTools

  // If this tool has a parent tool (hierarchical nesting), try to attach to it
  if (parentToolUseId) {
    let foundParent = false

    // Check if parent is still pending (streaming, not finished yet)
    if (state.pendingToolCalls.has(parentToolUseId)) {
      const updatedPending = new Map(state.pendingChildren)
      const pending = updatedPending.get(parentToolUseId) || []
      updatedPending.set(parentToolUseId, [...pending, {
        id: crypto.randomUUID(),
        role: 'tool',
        toolCallId: toolCallId,
        content: '',
        toolCalls: [completedToolCall],
      } as PlatformMessage])
      state.pendingChildren = updatedPending
      if (state.currentToolCall?.id === toolCallId) {
        state.currentToolCall = null
      }
      return state
    }

    // Search for parent tool in messages
    for (let i = messages.length - 1; i >= 0; i--) {
      if (messages[i].toolCalls) {
        const parentToolIdx = messages[i].toolCalls!.findIndex(tc => tc.id === parentToolUseId)
        if (parentToolIdx !== -1) {
          const childExists = messages[i].toolCalls!.some(tc => tc.id === toolCallId)
          if (!childExists) {
            messages[i] = {
              ...messages[i],
              toolCalls: [...(messages[i].toolCalls || []), completedToolCall]
            }
          }
          foundParent = true
          break
        }
      }
    }

    if (foundParent) {
      state.messages = messages
      if (state.currentToolCall?.id === toolCallId) {
        state.currentToolCall = null
      }
      return state
    }
  }

  // Attach to the correct assistant message.
  // AG-UI spec: use parentMessageId to find the exact assistant message.
  // Fallback: search backwards for the last assistant message.
  let foundAssistant = false
  for (let i = messages.length - 1; i >= 0; i--) {
    const isTargetMessage = parentMessageId
      ? messages[i].id === parentMessageId
      : messages[i].role === 'assistant'

    if (isTargetMessage) {
      const existingToolCalls = messages[i].toolCalls || []

      if (existingToolCalls.some(tc => tc.id === toolCallId)) {
        foundAssistant = true
        break
      }

      const pendingForThisTool = state.pendingChildren.get(toolCallId) || []
      const childToolCalls = pendingForThisTool.flatMap(child => child.toolCalls || [])

      messages[i] = {
        ...messages[i],
        toolCalls: [...existingToolCalls, completedToolCall, ...childToolCalls]
      }

      if (pendingForThisTool.length > 0) {
        const updatedPending = new Map(state.pendingChildren)
        updatedPending.delete(toolCallId)
        state.pendingChildren = updatedPending
      }

      foundAssistant = true
      break
    }
  }

  // If target message not found, add as standalone tool message
  if (!foundAssistant) {
    const toolMessage = {
      id: crypto.randomUUID(),
      role: 'tool' as const,
      content: '',
      toolCallId: toolCallId,
      toolCalls: [completedToolCall],
      timestamp: String(event.timestamp ?? ''),
    } as PlatformMessage
    messages.push(toolMessage)
    callbacks.onMessage?.(toolMessage)
  }

  state.messages = messages
  state.currentToolCall = null
  return state
}

function handleToolCallResult(
  state: AGUIClientState,
  event: ToolCallResultEvent,
): AGUIClientState {
  const toolCallId = event.toolCallId
  const resultContent = event.content || ''
  if (!toolCallId) return state

  let found = false

  // Search in committed messages first
  const messages = [...state.messages]
  for (let i = messages.length - 1; i >= 0; i--) {
    if (messages[i].toolCalls) {
      const tcIdx = messages[i].toolCalls!.findIndex(tc => tc.id === toolCallId)
      if (tcIdx >= 0) {
        const updatedToolCalls = [...messages[i].toolCalls!]
        updatedToolCalls[tcIdx] = {
          ...updatedToolCalls[tcIdx],
          result: resultContent,
          status: 'completed',
        }
        messages[i] = { ...messages[i], toolCalls: updatedToolCalls }
        state.messages = messages
        found = true
        break
      }
    }
  }

  // If not found, search in pendingChildren
  if (!found && state.pendingChildren.size > 0) {
    const updatedPendingChildren = new Map(state.pendingChildren)
    for (const [parentId, children] of updatedPendingChildren) {
      for (let j = 0; j < children.length; j++) {
        if (children[j].toolCalls) {
          const tcIdx = children[j].toolCalls!.findIndex(tc => tc.id === toolCallId)
          if (tcIdx >= 0) {
            const updatedChildren = [...children]
            const updatedToolCalls = [...updatedChildren[j].toolCalls!]
            updatedToolCalls[tcIdx] = {
              ...updatedToolCalls[tcIdx],
              result: resultContent,
              status: 'completed',
            }
            updatedChildren[j] = { ...updatedChildren[j], toolCalls: updatedToolCalls }
            updatedPendingChildren.set(parentId, updatedChildren)
            state.pendingChildren = updatedPendingChildren
            found = true
            break
          }
        }
      }
      if (found) break
    }
  }

  return state
}

function handleStateDelta(
  state: AGUIClientState,
  event: StateDeltaEvent,
): AGUIClientState {
  const stateClone = { ...state.state }
  for (const patch of event.delta as Array<{ op: string; path: string; value?: unknown }>) {
    const key = patch.path.startsWith('/') ? patch.path.slice(1) : patch.path
    if (patch.op === 'add' || patch.op === 'replace') {
      stateClone[key] = patch.value
    } else if (patch.op === 'remove') {
      delete stateClone[key]
    }
  }
  state.state = stateClone
  return state
}

function handleMessagesSnapshot(
  state: AGUIClientState,
  event: MessagesSnapshotEvent,
  callbacks: EventHandlerCallbacks,
): AGUIClientState {
  // Filter out hidden messages from snapshot
  const visibleMessages = (event.messages as PlatformMessage[]).filter(msg => {
    const isHidden = callbacks.hiddenMessageIdsRef.current.has(msg.id)
    return !isHidden
  })

  // Normalize snapshot: reconstruct parent-child tool call hierarchy
  const normalizedMessages = normalizeSnapshotMessages(visibleMessages)

  // Merge normalized snapshot into existing messages while preserving
  // chronological order.
  const snapshotMap = new Map(normalizedMessages.map(m => [m.id, m]))
  const existingIds = new Set(state.messages.map(m => m.id))

  // Update existing messages in-place with snapshot data.
  const merged: PlatformMessage[] = state.messages.map(msg => {
    const snapshotVersion = snapshotMap.get(msg.id)
    if (!snapshotVersion) return msg

    // For assistant messages, merge toolCalls to preserve streaming data
    if (msg.role === 'assistant' && msg.toolCalls?.length && snapshotVersion.toolCalls?.length) {
      const mergedToolCalls = [...snapshotVersion.toolCalls]
      for (const existingTC of msg.toolCalls) {
        const snapshotTC = mergedToolCalls.find(tc => tc.id === existingTC.id)
        if (snapshotTC) {
          if (existingTC.function.name && existingTC.function.name !== 'tool' &&
              (!snapshotTC.function.name || snapshotTC.function.name === 'tool')) {
            (snapshotTC as PlatformToolCall).function = {
              ...snapshotTC.function,
              name: existingTC.function.name,
            }
          }
          if (existingTC.function.arguments && !snapshotTC.function.arguments) {
            (snapshotTC as PlatformToolCall).function = {
              ...snapshotTC.function,
              arguments: existingTC.function.arguments,
            }
          }
          if (existingTC.parentToolUseId && !snapshotTC.parentToolUseId) {
            (snapshotTC as PlatformToolCall).parentToolUseId = existingTC.parentToolUseId
          }
        } else {
          mergedToolCalls.push(existingTC)
        }
      }
      return { ...snapshotVersion, toolCalls: mergedToolCalls }
    }

    return snapshotVersion
  })

  // Insert new snapshot messages at the correct position
  for (let i = 0; i < normalizedMessages.length; i++) {
    const msg = normalizedMessages[i]
    if (existingIds.has(msg.id)) continue

    let insertBeforeId: string | null = null
    for (let j = i + 1; j < normalizedMessages.length; j++) {
      if (existingIds.has(normalizedMessages[j].id)) {
        insertBeforeId = normalizedMessages[j].id
        break
      }
    }

    if (insertBeforeId) {
      const idx = merged.findIndex(m => m.id === insertBeforeId)
      if (idx >= 0) {
        merged.splice(idx, 0, msg)
      } else {
        merged.push(msg)
      }
    } else {
      merged.push(msg)
    }
    existingIds.add(msg.id)
  }

  // Recover tool names from streaming state before cleanup
  const toolNameMap = new Map<string, string>()
  for (const msg of merged) {
    if (msg.role === 'tool' && msg.toolCalls) {
      for (const tc of msg.toolCalls) {
        if (tc.id && tc.function.name && tc.function.name !== 'tool' && tc.function.name !== 'unknown_tool') {
          toolNameMap.set(tc.id, tc.function.name)
        }
      }
    }
  }
  for (const [id, pending] of state.pendingToolCalls) {
    if (pending.name && pending.name !== 'tool' && pending.name !== 'unknown_tool') {
      toolNameMap.set(id, pending.name)
    }
  }
  // Apply recovered names
  for (const msg of merged) {
    if (msg.role === 'assistant' && msg.toolCalls) {
      for (const tc of msg.toolCalls) {
        if ((!tc.function.name || tc.function.name === 'tool' || tc.function.name === 'unknown_tool') &&
            toolNameMap.has(tc.id)) {
          (tc as PlatformToolCall).function = {
            ...tc.function,
            name: toolNameMap.get(tc.id)!,
          }
        }
      }
    }
  }

  // Remove redundant standalone role=tool messages that are now nested
  const nestedToolCallIds = new Set<string>()
  for (const msg of merged) {
    if (msg.role === 'assistant' && msg.toolCalls) {
      for (const tc of msg.toolCalls) {
        nestedToolCallIds.add(tc.id)
      }
    }
  }
  state.messages = merged.filter(msg => {
    if (msg.role !== 'tool') return true
    if ('toolCallId' in msg && msg.toolCallId && nestedToolCallIds.has(msg.toolCallId)) return false
    if (msg.toolCalls?.some(tc => nestedToolCallIds.has(tc.id))) return false
    return true
  })

  // Clear pendingChildren -- the normalized snapshot subsumes any
  // pending child data from streaming
  state.pendingChildren = new Map()
  return state
}

function handleActivitySnapshot(
  state: AGUIClientState,
  event: PlatformActivitySnapshotEvent,
): AGUIClientState {
  if (event.activities) {
    state.activities = event.activities
  }
  return state
}

function handleActivityDelta(
  state: AGUIClientState,
  event: PlatformActivityDeltaEvent,
): AGUIClientState {
  if (!event.delta) return state

  const activitiesClone = [...state.activities]
  for (const patch of event.delta) {
    if (patch.op === 'add') {
      activitiesClone.push(patch.activity)
    } else if (patch.op === 'update') {
      const idx = activitiesClone.findIndex((a) => a.id === patch.activity.id)
      if (idx >= 0) {
        activitiesClone[idx] = patch.activity
      }
    } else if (patch.op === 'remove') {
      const idx = activitiesClone.findIndex((a) => a.id === patch.activity.id)
      if (idx >= 0) {
        activitiesClone.splice(idx, 1)
      }
    }
  }
  state.activities = activitiesClone
  return state
}

function handleStepStarted(
  state: AGUIClientState,
  event: StepStartedEvent,
): AGUIClientState {
  state.state = {
    ...state.state,
    currentStep: {
      id: event.stepName,
      name: event.stepName,
      status: 'running',
    },
  }
  return state
}

function handleStepFinished(state: AGUIClientState): AGUIClientState {
  const stateClone = { ...state.state }
  delete stateClone.currentStep
  state.state = stateClone
  return state
}

// ── Reasoning event handlers ──

function handleReasoningMessageStart(
  state: AGUIClientState,
  event: ReasoningMessageStartEvent,
): AGUIClientState {
  state.currentReasoning = {
    id: event.messageId || null,
    content: '',
    timestamp: String(event.timestamp ?? ''),
  }
  return state
}

function handleReasoningMessageContent(
  state: AGUIClientState,
  event: ReasoningMessageContentEvent,
): AGUIClientState {
  if (state.currentReasoning) {
    state.currentReasoning = {
      ...state.currentReasoning,
      content: (state.currentReasoning.content || '') + event.delta,
    }
  }
  return state
}

function handleReasoningMessageEnd(
  state: AGUIClientState,
  event: ReasoningMessageEndEvent,
  callbacks: EventHandlerCallbacks,
): AGUIClientState {
  if (state.currentReasoning?.content) {
    const msg = {
      id: state.currentReasoning.id || crypto.randomUUID(),
      role: 'assistant' as const,
      content: state.currentReasoning.content,
      metadata: { type: 'reasoning_block' },
      timestamp: String(event.timestamp ?? ''),
    } as PlatformMessage
    state.messages = [...state.messages, msg]
    callbacks.onMessage?.(msg)
  }
  state.currentReasoning = null
  return state
}

// ── RAW and META event handlers ──

function handleRawEvent(
  state: AGUIClientState,
  event: PlatformRawEvent,
  callbacks: EventHandlerCallbacks,
): AGUIClientState {
  const rawData = event.event || event.data

  // Handle message metadata (for hiding auto-sent messages)
  if (rawData?.type === 'message_metadata' && rawData?.hidden) {
    const messageId = rawData.messageId as string
    if (messageId) {
      callbacks.hiddenMessageIdsRef.current.add(messageId)
      state.messages = state.messages.filter(m => m.id !== messageId)
    }
    return state
  }

  // Handle Langfuse trace_id for feedback association
  if (rawData?.type === 'langfuse_trace' && rawData?.traceId) {
    const traceId = rawData.traceId as string
    callbacks.onTraceId?.(traceId)
    return state
  }

  // Handle thinking blocks from Claude SDK
  if (rawData?.type === 'thinking_block') {
    const msg = {
      id: crypto.randomUUID(),
      role: 'assistant' as const,
      content: String(rawData.thinking ?? ''),
      metadata: {
        type: 'thinking_block',
        thinking: String(rawData.thinking ?? ''),
        signature: String(rawData.signature ?? ''),
      },
      timestamp: String(event.timestamp ?? ''),
    } as PlatformMessage
    state.messages = [...state.messages, msg]
    callbacks.onMessage?.(msg)
    return state
  }

  // Handle user message echoes from backend
  if (rawData?.role === 'user' && rawData?.content) {
    const messageId = String(rawData.id ?? '') || crypto.randomUUID()
    const exists = state.messages.some(m => m.id === messageId)
    const isHidden = callbacks.hiddenMessageIdsRef.current.has(messageId)
    if (!exists && !isHidden) {
      const msg = {
        id: messageId,
        role: 'user' as const,
        content: String(rawData.content),
        timestamp: String(event.timestamp ?? ''),
      } as PlatformMessage
      state.messages = [...state.messages, msg]
      callbacks.onMessage?.(msg)
    }
    return state
  }

  // Handle other message data
  if (rawData?.role && rawData?.content) {
    const msg = {
      id: String(rawData.id ?? '') || crypto.randomUUID(),
      role: String(rawData.role),
      content: String(rawData.content),
      timestamp: String(event.timestamp ?? ''),
    } as PlatformMessage
    state.messages = [...state.messages, msg]
    callbacks.onMessage?.(msg)
  }
  return state
}

function handleMetaEvent(
  state: AGUIClientState,
  event: AGUIMetaEvent,
): AGUIClientState {
  const metaType = event.metaType
  const messageId = event.payload?.messageId as string | undefined

  if (messageId && (metaType === 'thumbs_up' || metaType === 'thumbs_down')) {
    const feedbackMap = new Map(state.messageFeedback)
    feedbackMap.set(messageId, metaType)
    state.messageFeedback = feedbackMap
  }
  return state
}

// ── Thinking event handlers (custom streaming thinking blocks) ──

function handleThinkingMessageStart(
  state: AGUIClientState,
): AGUIClientState {
  state.currentThinking = {
    id: crypto.randomUUID(),
    content: '',
    timestamp: new Date().toISOString(),
  }
  return state
}

function handleThinkingMessageContent(
  state: AGUIClientState,
  event: { type: string; delta: string },
): AGUIClientState {
  if (state.currentThinking) {
    state.currentThinking = {
      ...state.currentThinking,
      content: (state.currentThinking.content || '') + event.delta,
    }
  }
  return state
}

function handleThinkingMessageEnd(
  state: AGUIClientState,
  event: { type: string; timestamp?: number },
  callbacks: EventHandlerCallbacks,
): AGUIClientState {
  if (state.currentThinking?.content) {
    const thinkingText = state.currentThinking.content
    const msg = {
      id: state.currentThinking.id || crypto.randomUUID(),
      role: 'assistant' as const,
      content: thinkingText,
      metadata: {
        type: 'thinking_block',
        thinking: thinkingText,
        signature: '',
      },
      timestamp: String(event.timestamp ?? state.currentThinking.timestamp ?? ''),
    } as PlatformMessage
    state.messages = [...state.messages, msg]
    callbacks.onMessage?.(msg)
  }
  state.currentThinking = null
  return state
}
