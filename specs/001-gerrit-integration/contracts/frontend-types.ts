/**
 * Frontend TypeScript types for the Gerrit Integration.
 * These extend the existing IntegrationsStatus type.
 */

// --- API Request/Response Types ---

export type GerritAuthMethod = 'http_basic' | 'git_cookies'

export interface GerritConnectRequest {
  instanceName: string
  url: string
  authMethod: GerritAuthMethod
  username?: string       // Required when authMethod is 'http_basic'
  httpToken?: string      // Required when authMethod is 'http_basic'
  gitcookiesContent?: string // Required when authMethod is 'git_cookies'
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
  url?: string
  authMethod?: GerritAuthMethod
  updatedAt?: string
}

export interface GerritTestResponse {
  valid: boolean
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
