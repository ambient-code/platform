import { apiClient } from './client'

export type IntegrationsStatus = {
  github: {
    installed: boolean
    installationId?: number
    githubUserId?: string
    host?: string
    updatedAt?: string
    pat: {
      configured: boolean
      updatedAt?: string
      valid?: boolean
    }
    active?: 'app' | 'pat'
  }
  google: {
    connected: boolean
    email?: string
    expiresAt?: string
    updatedAt?: string
    valid?: boolean
  }
  jira: {
    connected: boolean
    url?: string
    email?: string
    updatedAt?: string
    valid?: boolean
  }
  gitlab: {
    connected: boolean
    instanceUrl?: string
    updatedAt?: string
    valid?: boolean
  }
}

/**
 * Get unified status for all integrations
 */
export async function getIntegrationsStatus(): Promise<IntegrationsStatus> {
  return apiClient.get<IntegrationsStatus>('/auth/integrations/status')
}
