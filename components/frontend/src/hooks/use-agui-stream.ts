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
import type { PlatformEvent, PlatformMessage } from '@/types/agui'
import { processAGUIEvent } from './agui/event-handlers'
import type { EventHandlerCallbacks } from './agui/event-handlers'
import { initialState } from './agui/types'
import type { UseAGUIStreamOptions, UseAGUIStreamReturn } from './agui/types'
import { frontendTools, executeFrontendTool } from '@/lib/frontend-tools'
import { sessionEventsAdapter } from '@/services/adapters/session-events'
import type { SessionEventsPort } from '@/services/ports/session-events'

// Re-export types so existing consumers can import from this module
export { initialState } from './agui/types'
export type { UseAGUIStreamOptions, UseAGUIStreamReturn } from './agui/types'

export function useAGUIStream(options: UseAGUIStreamOptions, port: SessionEventsPort = sessionEventsAdapter): UseAGUIStreamReturn {
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

  const [state, setState] = useState(initialState)
  const [isRunActive, setIsRunActive] = useState(false)
  const currentRunIdRef = useRef<string | null>(null)
  const eventSourceRef = useRef<EventSource | null>(null)
  const reconnectTimeoutRef = useRef<NodeJS.Timeout | null>(null)
  const reconnectAttemptsRef = useRef(0)
  const mountedRef = useRef(false)
  // Store latest processEvent to avoid stale closure in EventSource handler
  const processEventRef = useRef<((event: PlatformEvent) => void) | null>(null)

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

  // Execute frontend tool calls
  const handleFrontendToolCall = useCallback(
    async (toolName: string, args: Record<string, unknown>): Promise<string> => {
      return executeFrontendTool(toolName, args, {
        projectName,
        sessionName,
      })
    },
    [projectName, sessionName],
  )

  // Process incoming AG-UI events
  const processEvent = useCallback(
    (event: PlatformEvent) => {
      onEvent?.(event)

      const callbacks: EventHandlerCallbacks = {
        onMessage,
        onError,
        onTraceId,
        setIsRunActive,
        currentRunIdRef,
        hiddenMessageIdsRef,
        onFrontendToolCall: handleFrontendToolCall,
      }

      setState((prev) => processAGUIEvent(prev, event, callbacks))
    },
    [onEvent, onMessage, onError, onTraceId, handleFrontendToolCall],
  )

  // Keep processEventRef up to date to avoid stale closures in EventSource handler
  useEffect(() => {
    processEventRef.current = processEvent
  }, [processEvent])

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

      const eventSource = port.createEventSource(projectName, sessionName, runId)
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
          const event = JSON.parse(e.data) as PlatformEvent
          // Use ref to avoid stale closure - ensures we always call the latest processEvent
          processEventRef.current?.(event)
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
    [projectName, sessionName, port, onConnected, onError, onDisconnected],
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
        await port.interrupt(projectName, sessionName, runId)
        setIsRunActive(false)
        currentRunIdRef.current = null
      } catch (error) {
        console.error('[useAGUIStream] Interrupt failed:', error)
        throw error
      }
    },
    [projectName, sessionName, port],
  )

  // Send a message to start/continue the conversation
  // AG-UI server pattern: POST returns SSE stream directly
  const sendMessage = useCallback(
    async (content: string, metadata?: Record<string, unknown>) => {
      const userMessage = {
        id: crypto.randomUUID(),
        role: 'user' as const,
        content,
        ...(metadata ? { metadata } : {}),
      }

      const userMsgWithTimestamp = {
        ...userMessage,
        timestamp: new Date().toISOString(),
      } as PlatformMessage
      setState((prev) => ({
        ...prev,
        status: 'connected',
        error: null,
        messages: [...prev.messages, userMsgWithTimestamp],
      }))

      try {
        const result = await port.sendMessage(projectName, sessionName, {
          threadId: state.threadId || sessionName,
          parentRunId: state.runId,
          messages: [userMessage],
          tools: frontendTools,
        })

        if (result.runId) {
          currentRunIdRef.current = result.runId
          setIsRunActive(true)
        }

        if (!eventSourceRef.current) {
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
    [projectName, sessionName, state.threadId, state.runId, connect, port],
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
    isStreaming: state.currentMessage !== null || state.currentToolCall !== null || state.pendingToolCalls.size > 0 || state.currentReasoning !== null,
    isRunActive,
  }
}
