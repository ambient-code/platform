export type SessionPhase =
  | 'Pending'
  | 'Creating'
  | 'Running'
  | 'Stopping'
  | 'Completed'
  | 'Failed'
  | 'Stopped'

export type DomainSession = {
  id: string
  name: string
  phase: SessionPhase
  agentId: string | null
  agentName: string | null
  projectId: string | null
  model: string | null
  startTime: string | null
  completionTime: string | null
  createdAt: string
  updatedAt: string
  annotations: Record<string, string>
}

export type DomainProject = {
  id: string
  name: string
  description: string | null
  status: string | null
  createdAt: string
  updatedAt: string
}

export type PaginatedResult<T> = {
  items: T[]
  total: number
  page: number
  size: number
  hasMore: boolean
}

export type ListParams = {
  page?: number
  size?: number
  search?: string
  orderBy?: string
}

export type SessionEventType =
  | 'user'
  | 'assistant'
  | 'text'
  | 'tool_use'
  | 'tool_result'
  | 'error'
  | 'lifecycle'
  | 'user_feedback'
  | 'system'

export type DomainSessionMessage = {
  id: string
  sessionId: string
  eventType: SessionEventType
  payload: string
  seq: number
  createdAt: string
}

export type FeedbackItem = {
  id: string
  type: 'element' | 'region'
  comment: string
  position: { x: number; y: number }
  dimensions?: { width: number; height: number }
  capturedHtml?: string
  viewportWidth: number
  viewportHeight: number
  deviceSize: string
  timestamp: string
}

export type FeedbackBatch = {
  items: FeedbackItem[]
  sessionId: string
  previewUrl: string
}
