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

// Re-export types so existing consumers can import from this module
export { initialState } from './agui/types'
export type { UseAGUIStreamOptions, UseAGUIStreamReturn } from './agui/types'

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

  const [state, setState] = useState(initialState)
  const [isRunActive, setIsRunActive] = useState(false)
  const currentRunIdRef = useRef<string | null>(null)
  const eventSourceRef = useRef<EventSource | null>(null)
  const reconnectTimeoutRef = useRef<NodeJS.Timeout | null>(null)
  const reconnectAttemptsRef = useRef(0)
  const mountedRef = useRef(false)

  // Event batching: buffer SSE events and flush via requestAnimationFrame
  // This prevents re-rendering on every token during streaming (~20-50 events/sec → 1 render/frame)
  const eventBufferRef = useRef<PlatformEvent[]>([])
  const rafIdRef = useRef<number | null>(null)

  // Refs for stable sendMessage (avoids recreating callback on every state change)
  const stateThreadIdRef = useRef<string | null>(null)
  const stateRunIdRef = useRef<string | null>(null)
  const stateStatusRef = useRef<string>('idle')


  // Exponential backoff config for reconnection
  const MAX_RECONNECT_DELAY = 30000 // 30 seconds max
  const BASE_RECONNECT_DELAY = 1000 // 1 second base

  // Track mounted state without causing re-renders
  useEffect(() => {
    mountedRef.current = true
    return () => {
      mountedRef.current = false
      // Cancel any pending rAF flush on unmount
      if (rafIdRef.current !== null) {
        cancelAnimationFrame(rafIdRef.current)
        rafIdRef.current = null
      }
    }
  }, [])

  // Keep sendMessage refs in sync with state
  useEffect(() => {
    stateThreadIdRef.current = state.threadId
    stateRunIdRef.current = state.runId
    stateStatusRef.current = state.status
  }, [state.threadId, state.runId, state.status])

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
      }

      setState((prev) => processAGUIEvent(prev, event, callbacks))
    },
    [onEvent, onMessage, onError, onTraceId],
  )

  // Keep a ref to processEvent for use inside rAF callbacks (avoids stale closures)
  const processEventRef = useRef(processEvent)
  useEffect(() => {
    processEventRef.current = processEvent
  }, [processEvent])

  // Flush all buffered events in a single synchronous pass
  // React 18 batches all setState calls within the same synchronous callback,
  // so N events → N setState calls → 1 re-render (instead of N re-renders)
  const flushEventBuffer = useCallback(() => {
    rafIdRef.current = null
    const events = eventBufferRef.current
    if (events.length === 0) return
    eventBufferRef.current = []

    for (const event of events) {
      processEventRef.current(event)
    }
  }, [])

  // Schedule a flush on the next animation frame (~60fps max)
  const scheduleFlush = useCallback(() => {
    if (rafIdRef.current === null) {
      rafIdRef.current = requestAnimationFrame(flushEventBuffer)
    }
  }, [flushEventBuffer])

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
          const event = JSON.parse(e.data) as PlatformEvent
          // Buffer events and flush via requestAnimationFrame for batched rendering
          eventBufferRef.current.push(event)
          scheduleFlush()
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
    [projectName, sessionName, scheduleFlush, onConnected, onError, onDisconnected],
  )

  // Disconnect from the event stream
  const disconnect = useCallback(() => {
    // Cancel any pending rAF flush and clear buffered events
    if (rafIdRef.current !== null) {
      cancelAnimationFrame(rafIdRef.current)
      rafIdRef.current = null
    }
    eventBufferRef.current = []

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
  // Uses refs for threadId/runId/status so the callback is stable across state changes
  const sendMessage = useCallback(
    async (content: string) => {
      // Send to backend via run endpoint - this returns an SSE stream
      const runUrl = `/api/projects/${encodeURIComponent(projectName)}/agentic-sessions/${encodeURIComponent(sessionName)}/agui/run`

      const userMessage = {
        id: crypto.randomUUID(),
        role: 'user' as const,
        content,
      }

      // Add user message to state immediately for instant UI feedback.
      setState((prev) => ({
        ...prev,
        status: 'connected',
        error: null,
        messages: [...prev.messages, {
          ...userMessage,
          timestamp: new Date().toISOString(),
        } as PlatformMessage],
      }))


      try {
        const response = await fetch(runUrl, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
          },
          body: JSON.stringify({
            threadId: stateThreadIdRef.current || sessionName,
            parentRunId: stateRunIdRef.current,
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
        if (stateStatusRef.current !== 'connected') {
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
    [projectName, sessionName, connect],
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
    isStreaming: state.currentMessage !== null || state.currentToolCall !== null || state.pendingToolCalls.size > 0 || state.currentThinking !== null,
    isRunActive,
  }
}
