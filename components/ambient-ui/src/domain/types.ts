export type SessionPhase =
  | 'Pending'
  | 'Creating'
  | 'Running'
  | 'Stopping'
  | 'Completed'
  | 'Failed'
  | 'Stopped'

export type DomainRepo = {
  url: string
  branch: string | null
  name: string | null
  autoPush: boolean
}

export type ReconciledRepoStatus = 'Cloning' | 'Ready' | 'Failed'

export type DomainReconciledRepo = {
  url: string
  name: string | null
  status: ReconciledRepoStatus | null
  currentActiveBranch: string | null
  defaultBranch: string | null
  clonedAt: string | null
}

export type ConditionStatus = 'True' | 'False' | 'Unknown'

export type DomainCondition = {
  type: string
  status: ConditionStatus
  reason: string | null
  message: string | null
  lastTransitionTime: string | null
}

export type DomainSession = {
  id: string
  name: string
  phase: SessionPhase
  agentId: string | null
  agentName: string | null
  projectId: string | null
  model: string | null
  temperature: number | null
  maxTokens: number | null
  timeout: number | null
  workflowId: string | null
  prompt: string | null
  sdkRestartCount: number
  startTime: string | null
  completionTime: string | null
  createdAt: string
  updatedAt: string
  annotations: Record<string, string>
  labels: Record<string, string>
  environmentVariables: Record<string, string>
  repos: DomainRepo[]
  reconciledRepos: DomainReconciledRepo[]
  conditions: DomainCondition[]
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
