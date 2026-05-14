export type RunPayload = {
  threadId: string
  parentRunId?: string | null
  messages: Array<{ id: string; role: 'user'; content: string; metadata?: Record<string, unknown> }>
  tools: unknown[]
}

export type RunResponse = {
  runId?: string
}

export type SessionEventsPort = {
  createEventSource: (projectName: string, sessionName: string, runId?: string) => EventSource
  sendMessage: (projectName: string, sessionName: string, payload: RunPayload) => Promise<RunResponse>
  interrupt: (projectName: string, sessionName: string, runId: string) => Promise<void>
}
