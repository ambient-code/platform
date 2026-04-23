import { apiClient } from './client'

export type GerritAuthMethod = 'http_basic' | 'git_cookies'

// Discriminated union for connect request
export type GerritConnectRequest = {
  instanceName: string
  url: string
} & (
  | { authMethod: 'http_basic'; username: string; httpToken: string }
  | { authMethod: 'git_cookies'; gitcookiesContent: string }
)

// Test request (no instanceName needed)
export type GerritTestRequest = {
  url: string
} & (
  | { authMethod: 'http_basic'; username: string; httpToken: string }
  | { authMethod: 'git_cookies'; gitcookiesContent: string }
)

export type GerritTestResponse = {
  valid: boolean
  message?: string
  error?: string
}

export type GerritInstanceStatus = {
  connected: boolean
  instanceName: string
  url: string
  authMethod: GerritAuthMethod
  updatedAt?: string
}

export type GerritInstancesResponse = {
  instances: GerritInstanceStatus[]
}

/**
 * Get all Gerrit instances for the authenticated user
 */
export async function getGerritInstances(): Promise<GerritInstancesResponse> {
  return apiClient.get<GerritInstancesResponse>('/auth/gerrit/instances')
}

/**
 * Get status of a specific Gerrit instance
 */
export async function getGerritInstanceStatus(instanceName: string): Promise<GerritInstanceStatus> {
  return apiClient.get<GerritInstanceStatus>(`/auth/gerrit/${encodeURIComponent(instanceName)}/status`)
}

/**
 * Connect a Gerrit instance for the authenticated user
 */
export async function connectGerrit(data: GerritConnectRequest): Promise<void> {
  await apiClient.post<void, GerritConnectRequest>('/auth/gerrit/connect', data)
}

/**
 * Disconnect a Gerrit instance for the authenticated user
 */
export async function disconnectGerrit(instanceName: string): Promise<void> {
  await apiClient.delete<void>(`/auth/gerrit/${encodeURIComponent(instanceName)}/disconnect`)
}

/**
 * Test Gerrit connection credentials without saving
 */
export async function testGerritConnection(data: GerritTestRequest): Promise<GerritTestResponse> {
  return apiClient.post<GerritTestResponse, GerritTestRequest>('/auth/gerrit/test', data)
}
