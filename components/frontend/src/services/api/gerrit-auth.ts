import { apiClient } from './client'

export type GerritAuthMethod = 'http_basic' | 'git_cookies'

export type GerritConnectRequest = {
  instanceName: string
  url: string
  authMethod: GerritAuthMethod
  username?: string
  httpToken?: string
  gitcookiesContent?: string
}

export type GerritTestRequest = {
  url: string
  authMethod: GerritAuthMethod
  username?: string
  httpToken?: string
  gitcookiesContent?: string
}

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
  updatedAt: string
}

export type GerritInstancesResponse = {
  instances: GerritInstanceStatus[]
}

/** Connect a Gerrit instance by validating and storing credentials. */
export async function connectGerrit(data: GerritConnectRequest): Promise<void> {
  await apiClient.post<void, GerritConnectRequest>('/auth/gerrit/connect', data)
}

/** Test Gerrit credentials without storing them. */
export async function testGerritConnection(data: GerritTestRequest): Promise<GerritTestResponse> {
  return apiClient.post<GerritTestResponse, GerritTestRequest>('/auth/gerrit/test', data)
}

/** List all connected Gerrit instances for the current user. */
export async function getGerritInstances(): Promise<GerritInstancesResponse> {
  return apiClient.get<GerritInstancesResponse>('/auth/gerrit/instances')
}

/** Get connection status for a specific Gerrit instance. */
export async function getGerritInstanceStatus(instanceName: string): Promise<GerritInstanceStatus> {
  return apiClient.get<GerritInstanceStatus>(`/auth/gerrit/${instanceName}/status`)
}

/** Disconnect a Gerrit instance and remove stored credentials. */
export async function disconnectGerrit(instanceName: string): Promise<void> {
  await apiClient.delete<void>(`/auth/gerrit/${instanceName}/disconnect`)
}
