import type { SessionEventsPort, RunPayload } from '../ports/session-events'

export function createSessionEventsAdapter(): SessionEventsPort {
  return {
    createEventSource: (projectName, sessionName, runId) => {
      let url = `/api/projects/${encodeURIComponent(projectName)}/agentic-sessions/${encodeURIComponent(sessionName)}/agui/events`
      if (runId) {
        url += `?runId=${encodeURIComponent(runId)}`
      }
      return new EventSource(url)
    },

    sendMessage: async (projectName: string, sessionName: string, payload: RunPayload) => {
      const url = `/api/projects/${encodeURIComponent(projectName)}/agentic-sessions/${encodeURIComponent(sessionName)}/agui/run`
      const response = await fetch(url, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      })
      if (!response.ok) {
        const errorText = await response.text()
        throw new Error(`Failed to send message: ${errorText || response.statusText}`)
      }
      return response.json()
    },

    interrupt: async (projectName: string, sessionName: string, runId: string) => {
      const url = `/api/projects/${encodeURIComponent(projectName)}/agentic-sessions/${encodeURIComponent(sessionName)}/agui/interrupt`
      const response = await fetch(url, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ runId }),
      })
      if (!response.ok) {
        let errorText = ''
        try { errorText = await response.text() } catch { /* ignore */ }
        throw new Error(`Failed to interrupt: ${errorText || response.statusText}`)
      }
    },
  }
}

export const sessionEventsAdapter = createSessionEventsAdapter()
