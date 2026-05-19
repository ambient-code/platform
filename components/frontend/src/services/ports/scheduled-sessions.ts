import type { AgenticSession, ScheduledSession, CreateScheduledSessionRequest, UpdateScheduledSessionRequest } from './types'

export type ScheduledSessionsPort = {
  listScheduledSessions: (projectName: string) => Promise<ScheduledSession[]>
  getScheduledSession: (projectName: string, name: string) => Promise<ScheduledSession>
  createScheduledSession: (projectName: string, data: CreateScheduledSessionRequest) => Promise<ScheduledSession>
  updateScheduledSession: (projectName: string, name: string, data: UpdateScheduledSessionRequest) => Promise<ScheduledSession>
  deleteScheduledSession: (projectName: string, name: string) => Promise<void>
  suspendScheduledSession: (projectName: string, name: string) => Promise<ScheduledSession>
  resumeScheduledSession: (projectName: string, name: string) => Promise<ScheduledSession>
  triggerScheduledSession: (projectName: string, name: string) => Promise<{ name: string; namespace: string }>
  listScheduledSessionRuns: (projectName: string, name: string) => Promise<AgenticSession[]>
}
