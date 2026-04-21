import { apiClient } from './client'

export type CodeRabbitStatus = {
  connected: boolean
  updatedAt?: string
}

export type CodeRabbitConnectRequest = {
  apiKey: string
}

/**
 * Get CodeRabbit connection status for the authenticated user
 */
export async function getCodeRabbitStatus(): Promise<CodeRabbitStatus> {
  return apiClient.get<CodeRabbitStatus>('/auth/coderabbit/status')
}

/**
 * Connect CodeRabbit account for the authenticated user
 */
export async function connectCodeRabbit(data: CodeRabbitConnectRequest): Promise<void> {
  await apiClient.post<void, CodeRabbitConnectRequest>('/auth/coderabbit/connect', data)
}

/**
 * Disconnect CodeRabbit account for the authenticated user
 */
export async function disconnectCodeRabbit(): Promise<void> {
  await apiClient.delete<void>('/auth/coderabbit/disconnect')
}
