import { apiClient } from './client'

export type GitLabStatus = {
  connected: boolean
  instanceUrl?: string
  updatedAt?: string
}

export type GitLabConnectRequest = {
  personalAccessToken: string
  instanceUrl?: string
}

/**
 * Get GitLab connection status for the authenticated user
 */
export async function getGitLabStatus(): Promise<GitLabStatus> {
  return apiClient.get<GitLabStatus>('/auth/gitlab/status')
}

/**
 * Connect GitLab account for the authenticated user
 */
export async function connectGitLab(data: GitLabConnectRequest): Promise<void> {
  await apiClient.post<void, GitLabConnectRequest>('/auth/gitlab/connect', data)
}

/**
 * Disconnect GitLab account for the authenticated user
 */
export async function disconnectGitLab(): Promise<void> {
  await apiClient.delete<void>('/auth/gitlab/disconnect')
}
