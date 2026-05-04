export type { AgenticSession, AgenticSessionSpec, AgenticSessionStatus, AgenticSessionPhase, LLMSettings, SessionRepo, ReconciledRepo, ReconciledWorkflow, SessionCondition, CreateAgenticSessionRequest, CreateAgenticSessionResponse, StopAgenticSessionRequest, CloneAgenticSessionRequest, Message, ContentBlock, TextBlock, ReasoningBlock, ToolUseBlock, ToolResultBlock, UserMessage, AgentMessage, SystemMessage, ResultMessage, ToolUseMessages, StoredAgentStatus, BotAccountRef, ResourceOverrides } from '@/types/api/sessions'
export type { Project, ProjectStatus, CreateProjectRequest, UpdateProjectRequest, PermissionAssignment, PermissionRole, SubjectType } from '@/types/api/projects'
export type { PaginationParams, PaginatedResponse, ApiError } from '@/types/api/common'
export type { ApiClientError } from '@/types/api/common'
export type { ScheduledSession, CreateScheduledSessionRequest, UpdateScheduledSessionRequest } from '@/types/api/scheduled-sessions'
export type { GitHubStatus, GitHubFork, CreateForkRequest, PRDiff, CreatePRRequest, GitHubConnectRequest, ListBranchesResponse } from '@/types/api/github'
export type { LLMModel, ListModelsResponse } from '@/types/api/models'
export type { UserProfile } from '@/services/api/auth'
export type { ClusterInfo } from '@/services/api/cluster'
export type { LoadingTipsResponse } from '@/services/api/config'
export type { McpServer, McpTool, McpStatusResponse, PodEvent, PodEventsResponse, RepoStatus, ReposStatusResponse, SessionExportResponse, GoogleDriveFileResponse, CapabilitiesResponse } from '@/services/api/sessions'
export type { WorkspaceItem, GitMergeStatus, GitStatus } from '@/services/api/workspace'
export type { ProjectKey, CreateKeyRequest, CreateKeyResponse } from '@/services/api/keys'
export type { Secret, SecretList, SecretsConfig } from '@/services/api/secrets'
export type { OOTBWorkflow, WorkflowMetadataResponse, WorkflowCommand, WorkflowAgent, WorkflowConfig } from '@/services/api/workflows'
export type { RunnerType, RunnerTypeAuth } from '@/services/api/runner-types'
export type { FeatureToggle } from '@/services/api/feature-flags-admin'
export type { LDAPUser, LDAPGroup } from '@/services/api/ldap'
export type { IntegrationsStatus } from '@/services/api/integrations'
export type { GitLabStatus, GitLabConnectRequest } from '@/services/api/gitlab-auth'
export type { GoogleOAuthStatus, GoogleOAuthURLResponse } from '@/services/api/google-auth'
export type { GerritAuthMethod, GerritConnectRequest, GerritTestRequest, GerritTestResponse, GerritInstanceStatus, GerritInstancesResponse } from '@/services/api/gerrit-auth'
export type { JiraStatus, JiraConnectRequest } from '@/services/api/jira-auth'
export type { CodeRabbitStatus, CodeRabbitConnectRequest } from '@/services/api/coderabbit-auth'
export type { MCPServerStatus, MCPConnectRequest } from '@/services/api/mcp-credentials'
export type { IntegrationStatus } from '@/services/api/projects'
export type { TaskOutputResponse } from '@/types/background-task'
export type { MCPServersConfig } from '@/types/agentic-session'
export type ProjectAccess = {
  project: string
  allowed: boolean
  reason?: string
  userRole: import('@/types/api/projects').PermissionRole
}

export type PaginatedResult<T> = {
  items: T[]
  totalCount: number
  hasMore: boolean
  nextPage: (() => Promise<PaginatedResult<T>>) | undefined
}
