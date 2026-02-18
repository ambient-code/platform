/**
 * Types and initial state for the AG-UI streaming hook.
 */

import type {
  AGUIClientState,
  PlatformEvent,
  PlatformMessage,
} from '@/types/agui'

export type UseAGUIStreamOptions = {
  projectName: string
  sessionName: string
  runId?: string
  autoConnect?: boolean
  onEvent?: (event: PlatformEvent) => void
  onMessage?: (message: PlatformMessage) => void
  onError?: (error: string) => void
  onConnected?: () => void
  onDisconnected?: () => void
  onTraceId?: (traceId: string) => void  // Called when Langfuse trace_id is received
}

export type UseAGUIStreamReturn = {
  state: AGUIClientState
  connect: (runId?: string) => void
  disconnect: () => void
  sendMessage: (content: string) => Promise<void>
  interrupt: () => Promise<void>
  isConnected: boolean
  isStreaming: boolean
  isRunActive: boolean
}

export const initialState: AGUIClientState = {
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
  currentReasoning: null,
}
