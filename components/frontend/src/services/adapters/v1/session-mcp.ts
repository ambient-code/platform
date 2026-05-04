import * as sessionsApi from '../../api/sessions'
import type { SessionMcpPort } from '../../ports/session-mcp'

type SessionMcpApi = Pick<typeof sessionsApi, 'getMcpStatus' | 'updateSessionMcpServers'>

export function createSessionMcpAdapter(api: SessionMcpApi): SessionMcpPort {
  return {
    getMcpStatus: api.getMcpStatus,
    updateSessionMcpServers: api.updateSessionMcpServers,
  }
}

export const sessionMcpAdapter = createSessionMcpAdapter(sessionsApi)
