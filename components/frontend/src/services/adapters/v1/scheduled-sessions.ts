import * as scheduledApi from '../../api/scheduled-sessions'
import type { ScheduledSessionsPort } from '../../ports/scheduled-sessions'

type ScheduledApi = typeof scheduledApi

export function createScheduledSessionsAdapter(api: ScheduledApi): ScheduledSessionsPort {
  return {
    listScheduledSessions: api.listScheduledSessions,
    getScheduledSession: api.getScheduledSession,
    createScheduledSession: api.createScheduledSession,
    updateScheduledSession: api.updateScheduledSession,
    deleteScheduledSession: api.deleteScheduledSession,
    suspendScheduledSession: api.suspendScheduledSession,
    resumeScheduledSession: api.resumeScheduledSession,
    triggerScheduledSession: api.triggerScheduledSession,
    listScheduledSessionRuns: api.listScheduledSessionRuns,
  }
}

export const scheduledSessionsAdapter = createScheduledSessionsAdapter(scheduledApi)
