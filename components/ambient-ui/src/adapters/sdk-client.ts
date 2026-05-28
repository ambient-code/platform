import type { AmbientClientConfig } from 'ambient-sdk'
import { SessionAPI, ProjectAPI } from 'ambient-sdk'

// BFF proxy config: empty baseUrl produces relative URLs like /api/ambient/v1/sessions
// which hit the Next.js BFF proxy. No token needed — the proxy adds it server-side.
//
// We construct SessionAPI/ProjectAPI directly instead of using AmbientClient because
// the AmbientClient constructor rejects empty baseUrl. The API classes accept
// AmbientClientConfig directly without validation.
const bffConfig: AmbientClientConfig = {
  baseUrl: '',
}

let sessions: SessionAPI | null = null
let projects: ProjectAPI | null = null

export function getSessionAPI(): SessionAPI {
  if (!sessions) {
    sessions = new SessionAPI(bffConfig)
  }
  return sessions
}

export function getProjectAPI(): ProjectAPI {
  if (!projects) {
    projects = new ProjectAPI(bffConfig)
  }
  return projects
}

export function getConfig(): AmbientClientConfig {
  return bffConfig
}
