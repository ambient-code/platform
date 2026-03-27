/**
 * Frontend TypeScript types for the Gerrit Integration.
 * These extend the existing IntegrationsStatus type.
 */

// --- API Request/Response Types ---

export type GerritAuthMethod = 'http_basic' | 'git_cookies'

export type GerritConnectRequest =
  | {
      instanceName: string
      url: string
      authMethod: 'http_basic'
      username: string
      httpToken: string
    }
  | {
      instanceName: string
      url: string
      authMethod: 'git_cookies'
      gitcookiesContent: string
    }

export interface GerritConnectResponse {
  message: string
  instanceName: string
  url: string
  authMethod: GerritAuthMethod
}

export interface GerritInstanceStatus {
  connected: boolean
  instanceName: string
  url: string
  authMethod: GerritAuthMethod
  updatedAt: string
}

export interface GerritTestResponse {
  valid: boolean
  message?: string
  error?: string
}

export interface GerritInstancesListResponse {
  instances: GerritInstanceStatus[]
}

// --- Extension to existing IntegrationsStatus ---

// Add to existing IntegrationsStatus type:
// gerrit: {
//   instances: GerritInstanceStatus[]
// }
