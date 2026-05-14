import { apiClient, getApiBaseUrl } from './client'
import type { RunPayload, RunResponse } from '../ports/session-events'

function sessionPath(projectName: string, sessionName: string): string {
  return `/projects/${encodeURIComponent(projectName)}/agentic-sessions/${encodeURIComponent(sessionName)}`
}

export function createEventSource(
  projectName: string,
  sessionName: string,
  runId?: string,
): EventSource {
  let url = `${getApiBaseUrl()}${sessionPath(projectName, sessionName)}/agui/events`
  if (runId) {
    url += `?runId=${encodeURIComponent(runId)}`
  }
  return new EventSource(url)
}

export async function sendMessage(
  projectName: string,
  sessionName: string,
  payload: RunPayload,
): Promise<RunResponse> {
  return apiClient.post<RunResponse, RunPayload>(
    `${sessionPath(projectName, sessionName)}/agui/run`,
    payload,
  )
}

export async function interrupt(
  projectName: string,
  sessionName: string,
  runId: string,
): Promise<void> {
  await apiClient.post(
    `${sessionPath(projectName, sessionName)}/agui/interrupt`,
    { runId },
  )
}
