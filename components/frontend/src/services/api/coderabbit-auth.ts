import { apiClient } from './client'

export type CodeRabbitStatus = {
  connected: boolean
  updatedAt?: string
}

export type CodeRabbitConnectRequest = {
  apiKey: string
}

export async function getCodeRabbitStatus(): Promise<CodeRabbitStatus> {
  return apiClient.get<CodeRabbitStatus>('/auth/coderabbit/status')
}

export async function connectCodeRabbit(data: CodeRabbitConnectRequest): Promise<void> {
  await apiClient.post<void, CodeRabbitConnectRequest>('/auth/coderabbit/connect', data)
}

export async function disconnectCodeRabbit(): Promise<void> {
  await apiClient.delete<void>('/auth/coderabbit/disconnect')
}

export async function testCodeRabbitConnection(data: CodeRabbitConnectRequest): Promise<{ valid: boolean; error?: string }> {
  return apiClient.post<{ valid: boolean; error?: string }, CodeRabbitConnectRequest>('/auth/coderabbit/test', data)
}
