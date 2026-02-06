import { apiClient } from './client'

export type JiraStatus = {
  connected: boolean
  url?: string
  email?: string
  updatedAt?: string
}

export type JiraConnectRequest = {
  url: string
  email: string
  apiToken: string
}

/**
 * Get Jira connection status for the authenticated user
 */
export async function getJiraStatus(): Promise<JiraStatus> {
  return apiClient.get<JiraStatus>('/auth/jira/status')
}

/**
 * Connect Jira account for the authenticated user
 */
export async function connectJira(data: JiraConnectRequest): Promise<void> {
  await apiClient.post<void, JiraConnectRequest>('/auth/jira/connect', data)
}

/**
 * Disconnect Jira account for the authenticated user
 */
export async function disconnectJira(): Promise<void> {
  await apiClient.delete<void>('/auth/jira/disconnect')
}
