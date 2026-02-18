'use client'

/**
 * AG-UI Event Stream Hook
 * 
 * EventSource-based hook for consuming AG-UI events from the backend.
 * Uses the same-origin SSE proxy to bypass browser EventSource auth limitations.
 * 
 * Reference: https://docs.ag-ui.com/concepts/events
 * Reference: https://docs.ag-ui.com/concepts/messages
 */

import { useCallback, useEffect, useRef, useState } from 'react'
import {
  AGUIClientState,
  AGUIEvent,
  AGUIEventType,
  AGUIMessage,
  AGUIRole,
  AGUIStepStartedEvent,
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

type UseAGUIStreamOptions = {
  projectName: string
  sessionName: string
  runId?: string
  autoConnect?: boolean
  onEvent?: (event: AGUIEvent) => void
  onMessage?: (message: AGUIMessage) => void
  onError?: (error: string) => void
  onConnected?: () => void
  onDisconnected?: () => void
  onTraceId?: (traceId: string) => void  // Called when Langfuse trace_id is received
}

type UseAGUIStreamReturn = {
  state: AGUIClientState
  connect: (runId?: string) => void
  disconnect: () => void
  sendMessage: (content: string) => Promise<void>
  interrupt: () => Promise<void>
  isConnected: boolean
  isStreaming: boolean
  isRunActive: boolean
}

  const initialState: AGUIClientState = {
    threadId: null,
    runId: null,
    status: 'idle',
    messages: [],
    state: {},
    activities: [],
    currentMessage: null,
    currentToolCall: null,  // DEPRECATED: kept for backward compat
    pendingToolCalls: new Map(),  // NEW: tracks ALL in-progress tool calls
    pendingChildren: new Map(),
    error: null,
    messageFeedback: new Map(),  // Track feedback for messages
  }

export function useAGUIStream(options: UseAGUIStreamOptions): UseAGUIStreamReturn {
  // Track hidden message IDs (auto-sent initial/workflow prompts)
  const hiddenMessageIdsRef = useRef<Set<string>>(new Set())
  const {
    projectName,
    sessionName,
    runId: initialRunId,
    autoConnect = false,
    onEvent,
    onMessage,
    onError,
    onConnected,
    onDisconnected,
    onTraceId,
  } = options

  const [state, setState] = useState<AGUIClientState>(initialState)
  const [isRunActive, setIsRunActive] = useState(false)
  const currentRunIdRef = useRef<string | null>(null)
  const eventSourceRef = useRef<EventSource | null>(null)
  const reconnectTimeoutRef = useRef<NodeJS.Timeout | null>(null)
  const reconnectAttemptsRef = useRef(0)
  const mountedRef = useRef(false)
  
  // Exponential backoff config for reconnection
  const MAX_RECONNECT_DELAY = 30000 // 30 seconds max
  const BASE_RECONNECT_DELAY = 1000 // 1 second base
  
  // Track mounted state without causing re-renders
  useEffect(() => {
    mountedRef.current = true
    return () => {
      mountedRef.current = false
    }
  }, [])

  // Process incoming AG-UI events
  const processEvent = useCallback(
    (event: AGUIEvent) => {
      onEvent?.(event)

      setState((prev) => {
        const newState = { ...prev }

        if (isRunStartedEvent(event)) {
          newState.threadId = event.threadId
          newState.runId = event.runId
          newState.status = 'connected'
          newState.error = null
          
          // Track active run
          currentRunIdRef.current = event.runId
          setIsRunActive(true)
          
          return newState
        }

        if (isRunFinishedEvent(event)) {
          newState.status = 'completed'
          
          // Mark run as inactive
          if (currentRunIdRef.current === event.runId) {
            setIsRunActive(false)
            currentRunIdRef.current = null
          }
          
          // Flush any pending message
          if (newState.currentMessage?.content) {
            const msg: AGUIMessage = {
              id: newState.currentMessage.id || crypto.randomUUID(),
              role: newState.currentMessage.role || AGUIRole.ASSISTANT,
              content: newState.currentMessage.content,
              timestamp: event.timestamp,
            }
            newState.messages = [...newState.messages, msg]
            onMessage?.(msg)
          }
          newState.currentMessage = null
          return newState
        }

        if (isRunErrorEvent(event)) {
          newState.status = 'error'
          newState.error = event.error
          onError?.(event.error)
          
          // Mark run as inactive on error
          if (currentRunIdRef.current === event.runId) {
            setIsRunActive(false)
            currentRunIdRef.current = null
          }
          
          return newState
        }

        if (isTextMessageStartEvent(event)) {
          newState.currentMessage = {
            id: event.messageId || null,
            role: event.role,
            content: '',
            timestamp: event.timestamp,  // Capture timestamp from event
          }
          return newState
        }

        if (isTextMessageContentEvent(event)) {
          if (newState.currentMessage) {
            // Create a NEW object so React detects the change and re-renders
            newState.currentMessage = {
              ...newState.currentMessage,
              content: (newState.currentMessage.content || '') + event.delta,
            }
          }
          return newState
        }

        if (isTextMessageEndEvent(event)) {
          if (newState.currentMessage?.content) {
            const messageId = newState.currentMessage.id || crypto.randomUUID();
            
            // Skip hidden messages (auto-sent initial/workflow prompts)
            if (hiddenMessageIdsRef.current.has(messageId)) {
              newState.currentMessage = null;
              return newState;
            }
            
            // Check if this message already exists (e.g., from MESSAGES_SNAPSHOT)
            const existingIndex = newState.messages.findIndex(m => m.id === messageId);
            
            if (existingIndex >= 0) {
              // Message exists - update content if different (don't duplicate)
              const existingMsg = newState.messages[existingIndex];
              if (existingMsg.content !== newState.currentMessage.content) {
                const updatedMessages = [...newState.messages];
                updatedMessages[existingIndex] = {
                  ...existingMsg,
                  content: newState.currentMessage.content,
                };
                newState.messages = updatedMessages;
              }
            } else {
              // Message doesn't exist - create new
              const msg: AGUIMessage = {
                id: messageId,
                role: newState.currentMessage.role || AGUIRole.ASSISTANT,
                content: newState.currentMessage.content,
                timestamp: event.timestamp,
              }
              newState.messages = [...newState.messages, msg]
              onMessage?.(msg)
            }
          }
          newState.currentMessage = null
          // Don't clear currentToolCall - tool calls might come after TEXT_MESSAGE_END
          return newState
        }

        if (isToolCallStartEvent(event)) {
          // Runner's ag_ui.core uses snake_case: parent_tool_call_id
          const parentToolId = (event as unknown as { parent_tool_call_id?: string }).parent_tool_call_id;
          
          // Store in pendingToolCalls Map to support parallel tool calls
          const updatedPending = new Map(newState.pendingToolCalls);
          updatedPending.set(event.toolCallId, {
            id: event.toolCallId,
            name: event.toolCallName || 'unknown_tool',
            args: '',
            parentToolUseId: parentToolId,
            timestamp: event.timestamp,  // Capture timestamp from event
          });
          newState.pendingToolCalls = updatedPending;
          
          // Also update currentToolCall for backward compat (UI rendering)
          newState.currentToolCall = {
            id: event.toolCallId,
            name: event.toolCallName,
            args: '',
            parentToolUseId: parentToolId,
          }
          return newState
        }

        if (event.type === AGUIEventType.TOOL_CALL_ARGS) {
          const toolCallId = event.toolCallId;
          const existing = newState.pendingToolCalls.get(toolCallId);
          
          if (existing) {
            // Update the pending tool call in Map
            const updatedPending = new Map(newState.pendingToolCalls);
            updatedPending.set(toolCallId, {
              ...existing,
              args: (existing.args || '') + event.delta,
            });
            newState.pendingToolCalls = updatedPending;
          }
          
          // Also update currentToolCall for backward compat (if it's the same tool)
          if (newState.currentToolCall?.id === toolCallId) {
            newState.currentToolCall = {
              ...newState.currentToolCall,
              args: (newState.currentToolCall.args || '') + event.delta,
            }
          }
          return newState
        }

        if (isToolCallEndEvent(event)) {
          const toolCallId = event.toolCallId || newState.currentToolCall?.id || crypto.randomUUID()
          
          // Get tool info from pendingToolCalls Map (supports parallel tool calls)
          const pendingTool = newState.pendingToolCalls.get(toolCallId);
          const toolCallName = pendingTool?.name || newState.currentToolCall?.name || 'unknown_tool'
          const toolCallArgs = pendingTool?.args || newState.currentToolCall?.args || ''
          const parentToolUseId = pendingTool?.parentToolUseId || newState.currentToolCall?.parentToolUseId
          
          // Defense in depth: Check if this tool already exists (shouldn't happen with fixed backend)
          const toolAlreadyExists = newState.messages.some(msg => 
            msg.toolCalls?.some(tc => tc.id === toolCallId)
          );
          
          if (toolAlreadyExists) {
            console.warn(`[useAGUIStream] BACKEND BUG: Tool ${toolCallName} (${toolCallId.substring(0, 8)}) already exists, skipping duplicate`);
            // Remove from pending maps and return
            const updatedPendingTools = new Map(newState.pendingToolCalls);
            updatedPendingTools.delete(toolCallId);
            newState.pendingToolCalls = updatedPendingTools;
            if (newState.currentToolCall?.id === toolCallId) {
              newState.currentToolCall = null;
            }
            return newState;
          }
          
          // Create completed tool call
          const completedToolCall = {
            id: toolCallId,
            name: toolCallName,
            args: toolCallArgs,
            result: event.result,
            status: event.error ? 'error' as const : 'completed' as const,
            error: event.error,
            parentToolUseId: parentToolUseId,
          }
          
          const messages = [...newState.messages]
          
          // Remove from pendingToolCalls Map
          const updatedPendingTools = new Map(newState.pendingToolCalls);
          updatedPendingTools.delete(toolCallId);
          newState.pendingToolCalls = updatedPendingTools;
          
          // If this tool has a parent, try to attach to it
          if (parentToolUseId) {
            let foundParent = false
            
            // Check if parent is still pending (streaming, not finished yet)
            if (newState.pendingToolCalls.has(parentToolUseId)) {
              // Parent is still streaming - store as pending child
              const updatedPending = new Map(newState.pendingChildren);
              const pending = updatedPending.get(parentToolUseId) || []
              updatedPending.set(parentToolUseId, [...pending, {
                id: crypto.randomUUID(),
                role: AGUIRole.TOOL,
                toolCallId: toolCallId,
                name: toolCallName,
                content: event.result || event.error || '',
                toolCalls: [completedToolCall],
              }])
              newState.pendingChildren = updatedPending;
              if (newState.currentToolCall?.id === toolCallId) {
                newState.currentToolCall = null;
              }
              return newState
            }
            
            // Search for parent in messages
            for (let i = messages.length - 1; i >= 0; i--) {
              // Check if parent is in this message's toolCalls array
              if (messages[i].toolCalls) {
                const parentToolIdx = messages[i].toolCalls!.findIndex(tc => tc.id === parentToolUseId)
                if (parentToolIdx !== -1) {
                  // Found parent! Check if child already attached
                  const childExists = messages[i].toolCalls!.some(tc => tc.id === toolCallId);
                  if (!childExists) {
                    const existingToolCalls = messages[i].toolCalls || []
                    messages[i] = {
                      ...messages[i],
                      toolCalls: [...existingToolCalls, completedToolCall]
                    }
                  }
                  foundParent = true
                  break
                }
              }
            }
            
            if (foundParent) {
              newState.messages = messages
              if (newState.currentToolCall?.id === toolCallId) {
                newState.currentToolCall = null;
              }
              return newState
            }
            
            // Parent not found - will attach to assistant message below
            console.warn(`[useAGUIStream] Parent ${parentToolUseId.substring(0, 8)} not found for child ${toolCallName}, attaching to assistant`)
          }
          
          // This is either a top-level tool or parent wasn't found
          // Attach to last assistant message
          let foundAssistant = false
          for (let i = messages.length - 1; i >= 0; i--) {
            if (messages[i].role === AGUIRole.ASSISTANT) {
              const existingToolCalls = messages[i].toolCalls || []
              
              // Check if tool already exists in this message
              if (existingToolCalls.some(tc => tc.id === toolCallId)) {
                foundAssistant = true;
                break;
              }
              
              // If this tool just finished and has pending children, attach them all now!
              const pendingForThisTool = newState.pendingChildren.get(toolCallId) || []
              const childToolCalls = pendingForThisTool.flatMap(child => child.toolCalls || [])
              
              messages[i] = {
                ...messages[i],
                toolCalls: [...existingToolCalls, completedToolCall, ...childToolCalls]
              }
              
              if (pendingForThisTool.length > 0) {
                const updatedPending = new Map(newState.pendingChildren);
                updatedPending.delete(toolCallId);
                newState.pendingChildren = updatedPending;
              }
              
              foundAssistant = true
              break
            }
          }
          
          // If no assistant, add as standalone
          if (!foundAssistant) {
            const toolMessage: AGUIMessage = {
              id: crypto.randomUUID(),
              role: AGUIRole.TOOL,
              content: event.result || event.error || '',
              toolCallId: toolCallId,
              name: toolCallName,
              toolCalls: [completedToolCall],
              timestamp: event.timestamp,
            }
            messages.push(toolMessage)
          }
          
          newState.messages = messages
          newState.currentToolCall = null
          return newState
        }

        if (isStateSnapshotEvent(event)) {
          newState.state = event.state
          return newState
        }

        if (event.type === AGUIEventType.STATE_DELTA) {
          // Apply state patches
          const stateClone = { ...newState.state }
          for (const patch of event.delta) {
            const key = patch.path.startsWith('/') ? patch.path.slice(1) : patch.path
            if (patch.op === 'add' || patch.op === 'replace') {
              stateClone[key] = patch.value
            } else if (patch.op === 'remove') {
              delete stateClone[key]
            }
          }
          newState.state = stateClone
          return newState
        }

        if (isMessagesSnapshotEvent(event)) {
          
          // Filter out hidden messages from snapshot
          const visibleMessages = event.messages.filter(msg => {
            const isHidden = hiddenMessageIdsRef.current.has(msg.id)
            return !isHidden
          })
          
          // CRITICAL: Don't replace messages - merge snapshot with any in-progress streaming messages
          // Snapshot contains completed messages, but streaming might have started new messages
          // that aren't in the snapshot yet
          const snapshotIds = new Set(visibleMessages.map(m => m.id))
          const streamingMessages = newState.messages.filter(m => !snapshotIds.has(m.id))
          
          newState.messages = [...visibleMessages, ...streamingMessages]
          return newState
        }

        if (isActivitySnapshotEvent(event)) {
          newState.activities = event.activities
          return newState
        }

        if (event.type === AGUIEventType.ACTIVITY_DELTA) {
          const activitiesClone = [...newState.activities]
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
          newState.activities = activitiesClone
          return newState
        }

        // Handle STEP events
        if (event.type === AGUIEventType.STEP_STARTED) {
          // Track current step in state
          newState.state = {
            ...newState.state,
            currentStep: {
              id: (event as AGUIStepStartedEvent).stepId,
              name: (event as AGUIStepStartedEvent).stepName,
              status: 'running',
            },
          }
          return newState
        }

        if (event.type === AGUIEventType.STEP_FINISHED) {
          // Clear current step
          const stateClone = { ...newState.state }
          delete stateClone.currentStep
          newState.state = stateClone
          return newState
        }

        // Handle RAW events (may contain message data or thinking blocks)
        if (event.type === AGUIEventType.RAW) {
          // RAW events use "event" field (AG-UI standard), or "data" field (legacy)
          type RawEventData = { event?: Record<string, unknown>; data?: Record<string, unknown> }
          const rawEvent = event as unknown as RawEventData
          const rawData = rawEvent.event || rawEvent.data
          
          // Handle message metadata (for hiding auto-sent messages)
          if (rawData?.type === 'message_metadata' && rawData?.hidden) {
            const messageId = rawData.messageId as string
            if (messageId) {
              hiddenMessageIdsRef.current.add(messageId)
            }
            return newState
          }
          
          // Handle Langfuse trace_id for feedback association
          if (rawData?.type === 'langfuse_trace' && rawData?.traceId) {
            const traceId = rawData.traceId as string
            onTraceId?.(traceId)
            return newState
          }
          
          const actualRawData = rawData
          
          // Handle thinking blocks from Claude SDK
          if (actualRawData?.type === 'thinking_block') {
            const msg: AGUIMessage = {
              id: crypto.randomUUID(),
              role: AGUIRole.ASSISTANT,
              content: actualRawData.thinking as string || '',
              metadata: {
                type: 'thinking_block',
                thinking: actualRawData.thinking as string,
                signature: actualRawData.signature as string,
              },
              timestamp: event.timestamp,
            }
            newState.messages = [...newState.messages, msg]
            onMessage?.(msg)
            return newState
          }
          
          // Handle user message echoes from backend
          if (actualRawData?.role === 'user' && actualRawData?.content) {
            // Check if this message already exists to prevent duplicates
            const messageId = (actualRawData.id as string) || crypto.randomUUID()
            const exists = newState.messages.some(m => m.id === messageId)
            if (!exists) {
              const msg: AGUIMessage = {
                id: messageId,
                role: AGUIRole.USER,
                content: actualRawData.content as string,
                timestamp: event.timestamp,
              }
              newState.messages = [...newState.messages, msg]
              onMessage?.(msg)
            }
            return newState
          }
          
          // Handle other message data
          if (actualRawData?.role && actualRawData?.content) {
            const msg: AGUIMessage = {
              id: (actualRawData.id as string) || crypto.randomUUID(),
              role: actualRawData.role as AGUIMessage['role'],
              content: actualRawData.content as string,
              timestamp: event.timestamp,
            }
            newState.messages = [...newState.messages, msg]
            onMessage?.(msg)
          }
          return newState
        }

        // Handle META events (user feedback: thumbs_up / thumbs_down)
        if (event.type === AGUIEventType.META) {
          const metaType = event.metaType
          const messageId = event.payload?.messageId as string | undefined
          
          if (messageId && (metaType === 'thumbs_up' || metaType === 'thumbs_down')) {
            const feedbackMap = new Map(newState.messageFeedback)
            feedbackMap.set(messageId, metaType)
            newState.messageFeedback = feedbackMap
          }
          return newState
        }

        return newState
      })
    },
    [onEvent, onMessage, onError, onTraceId],
  )

  // Connect to the AG-UI event stream
  const connect = useCallback(
    (runId?: string) => {
      // Disconnect existing connection
      if (eventSourceRef.current) {
        eventSourceRef.current.close()
        eventSourceRef.current = null
      }

      setState((prev) => ({
        ...prev,
        status: 'connecting',
        error: null,
      }))

      // Build SSE URL through Next.js proxy
      let url = `/api/projects/${encodeURIComponent(projectName)}/agentic-sessions/${encodeURIComponent(sessionName)}/agui/events`
      if (runId) {
        url += `?runId=${encodeURIComponent(runId)}`
      }

      const eventSource = new EventSource(url)
      eventSourceRef.current = eventSource

      eventSource.onopen = () => {
        // Reset reconnect attempts on successful connection
        reconnectAttemptsRef.current = 0
        setState((prev) => ({
          ...prev,
          status: 'connected',
        }))
        onConnected?.()
      }

      eventSource.onmessage = (e) => {
        try {
          const event = JSON.parse(e.data) as AGUIEvent
          processEvent(event)
        } catch (err) {
          console.error('Failed to parse AG-UI event:', err)
        }
      }

      eventSource.onerror = () => {
        // IMPORTANT: Close the EventSource immediately to prevent browser's native reconnect
        // from firing alongside our custom reconnect logic
        eventSource.close()
        
        // Only proceed if this is still our active EventSource
        if (eventSourceRef.current !== eventSource) {
          return
        }
        eventSourceRef.current = null
        
        // Don't reconnect if component is unmounted
        if (!mountedRef.current) {
          return
        }
        
        setState((prev) => ({
          ...prev,
          status: 'error',
          error: 'Connection error',
        }))
        onError?.('Connection error')
        onDisconnected?.()

        // Clear any existing reconnect timeout
        if (reconnectTimeoutRef.current) {
          clearTimeout(reconnectTimeoutRef.current)
        }
        
        // Exponential backoff: 1s, 2s, 4s, 8s, 16s, 30s (max)
        reconnectAttemptsRef.current++
        const delay = Math.min(
          BASE_RECONNECT_DELAY * Math.pow(2, reconnectAttemptsRef.current - 1),
          MAX_RECONNECT_DELAY
        )
        
        console.log(`[useAGUIStream] Reconnecting in ${delay}ms (attempt ${reconnectAttemptsRef.current})`)
        
        reconnectTimeoutRef.current = setTimeout(() => {
          if (mountedRef.current) {
            connect(runId)
          }
        }, delay)
      }
    },
    [projectName, sessionName, processEvent, onConnected, onError, onDisconnected],
  )

  // Disconnect from the event stream
  const disconnect = useCallback(() => {
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current)
      reconnectTimeoutRef.current = null
    }
    if (eventSourceRef.current) {
      eventSourceRef.current.close()
      eventSourceRef.current = null
    }
    setState((prev) => ({
      ...prev,
      status: 'idle',
    }))
    setIsRunActive(false)
    currentRunIdRef.current = null
    onDisconnected?.()
  }, [onDisconnected])

  // Interrupt the current run (stop Claude mid-execution)
  const interrupt = useCallback(
    async () => {
      const runId = currentRunIdRef.current
      if (!runId) {
        console.warn('[useAGUIStream] No active run to interrupt')
        return
      }

      try {
        const interruptUrl = `/api/projects/${encodeURIComponent(projectName)}/agentic-sessions/${encodeURIComponent(sessionName)}/agui/interrupt`

        const response = await fetch(interruptUrl, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ runId }),
        })

        if (!response.ok) {
          throw new Error(`Failed to interrupt: ${response.statusText}`)
        }
        
        // Mark run as inactive immediately (backend will send RUN_FINISHED or RUN_ERROR)
        setIsRunActive(false)
        currentRunIdRef.current = null
        
      } catch (error) {
        console.error('[useAGUIStream] Interrupt failed:', error)
        throw error
      }
    },
    [projectName, sessionName],
  )

  // Send a message to start/continue the conversation
  // AG-UI server pattern: POST returns SSE stream directly
  const sendMessage = useCallback(
    async (content: string) => {
      // Set status to connected when starting a new message
      setState((prev) => ({
        ...prev,
        status: 'connected',
        error: null,
      }))

      // Send to backend via run endpoint - this returns an SSE stream
      const runUrl = `/api/projects/${encodeURIComponent(projectName)}/agentic-sessions/${encodeURIComponent(sessionName)}/agui/run`

      const userMessage = {
        id: crypto.randomUUID(),
        role: AGUIRole.USER,
        content,
      }


      try {
        const response = await fetch(runUrl, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
          },
          body: JSON.stringify({
            threadId: state.threadId || sessionName,
            parentRunId: state.runId,
            messages: [userMessage],
          }),
        })

        if (!response.ok) {
          const errorText = await response.text()
          console.error(`[useAGUIStream] /agui/run error: ${errorText}`)
          setState((prev) => ({
            ...prev,
            status: 'error',
            error: errorText,
          }))
          setIsRunActive(false)
          throw new Error(`Failed to send message: ${errorText}`)
        }

        // AG-UI middleware pattern: POST creates run and returns metadata immediately
        // Events are broadcast to GET /agui/events subscribers (avoid concurrent streams)
        const result = await response.json()
        
        // Mark run as active and track runId
        if (result.runId) {
          currentRunIdRef.current = result.runId
          setIsRunActive(true)
        }
        
        // Ensure we're connected to the thread stream to receive events
        if (state.status !== 'connected') {
          connect()
        }
      } catch (error) {
        console.error(`[useAGUIStream] sendMessage error:`, error)
        setState((prev) => ({
          ...prev,
          status: 'error',
          error: error instanceof Error ? error.message : 'Unknown error',
        }))
        throw error
      }
    },
    [projectName, sessionName, state.threadId, state.runId, state.status, connect],
  )

  // Auto-connect on mount if enabled (client-side only)
  const autoConnectAttemptedRef = useRef(false)
  useEffect(() => {
    if (typeof window === 'undefined') return // Skip during SSR
    if (autoConnectAttemptedRef.current) return // Only auto-connect once
    
    if (autoConnect && mountedRef.current) {
      autoConnectAttemptedRef.current = true
      connect(initialRunId)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [autoConnect])

  return {
    state,
    connect,
    disconnect,
    sendMessage,
    interrupt,
    isConnected: state.status === 'connected',
    isStreaming: state.currentMessage !== null || state.currentToolCall !== null || state.pendingToolCalls.size > 0,
    isRunActive,
  }
}

