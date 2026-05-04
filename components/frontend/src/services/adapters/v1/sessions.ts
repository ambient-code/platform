import * as sessionsApi from '../../api/sessions'
import type { SessionsPort } from '../../ports/sessions'
import { toPaginatedResult } from '../pagination'

type SessionsApi = typeof sessionsApi

export function createSessionsAdapter(api: SessionsApi): SessionsPort {
  return {
    listSessions: async (projectName, params = {}) => {
      const response = await api.listSessionsPaginated(projectName, params)
      return toPaginatedResult(response, (p) => api.listSessionsPaginated(projectName, p))
    },
    getSession: api.getSession,
    createSession: api.createSession,
    stopSession: api.stopSession,
    startSession: api.startSession,
    cloneSession: api.cloneSession,
    deleteSession: api.deleteSession,
    getSessionPodEvents: api.getSessionPodEvents,
    updateSessionDisplayName: api.updateSessionDisplayName,
    getSessionExport: api.getSessionExport,
    switchSessionModel: api.switchSessionModel,
    saveToGoogleDrive: api.saveToGoogleDrive,
  }
}

export const sessionsAdapter = createSessionsAdapter(sessionsApi)
