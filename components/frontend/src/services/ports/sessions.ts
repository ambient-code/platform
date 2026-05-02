import type { AgenticSession, CreateAgenticSessionRequest, StopAgenticSessionRequest, CloneAgenticSessionRequest, PaginationParams, PaginatedResult, PodEventsResponse, SessionExportResponse, GoogleDriveFileResponse } from './types'

export type SessionsPort = {
  listSessions: (projectName: string, params?: PaginationParams) => Promise<PaginatedResult<AgenticSession>>
  getSession: (projectName: string, sessionName: string) => Promise<AgenticSession>
  createSession: (projectName: string, data: CreateAgenticSessionRequest) => Promise<AgenticSession>
  stopSession: (projectName: string, sessionName: string, data?: StopAgenticSessionRequest) => Promise<string>
  startSession: (projectName: string, sessionName: string) => Promise<{ message: string }>
  cloneSession: (projectName: string, sessionName: string, data: CloneAgenticSessionRequest) => Promise<AgenticSession>
  deleteSession: (projectName: string, sessionName: string) => Promise<void>
  getSessionPodEvents: (projectName: string, sessionName: string) => Promise<PodEventsResponse>
  updateSessionDisplayName: (projectName: string, sessionName: string, displayName: string) => Promise<AgenticSession>
  getSessionExport: (projectName: string, sessionName: string) => Promise<SessionExportResponse>
  switchSessionModel: (projectName: string, sessionName: string, model: string) => Promise<AgenticSession>
  saveToGoogleDrive: (projectName: string, sessionName: string, content: string, filename: string, userEmail: string, serverName?: string) => Promise<GoogleDriveFileResponse>
}
