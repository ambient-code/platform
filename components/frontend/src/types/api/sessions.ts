/**
 * Agentic Session API types
 * These types align with the backend Go structs and Kubernetes CRD
 */

export type UserContext = {
  userId: string;
  displayName: string;
  groups: string[];
};

export type BotAccountRef = {
  name: string;
};

export type ResourceOverrides = {
  cpu?: string;
  memory?: string;
  storageClass?: string;
  priorityClass?: string;
};

export type AgenticSessionPhase =
  | 'Pending'
  | 'Creating'
  | 'Running'
  | 'Stopping'
  | 'Stopped'
  | 'Completed'
  | 'Failed';

export type LLMSettings = {
  model: string;
  temperature: number;
  maxTokens: number;
};

export type SessionRepo = {
  url: string;
  branch?: string;
  autoPush?: boolean;
};

export type AgenticSessionSpec = {
  initialPrompt?: string;
  llmSettings: LLMSettings;
  timeout: number;
  displayName?: string;
  project?: string;
  interactive?: boolean;
  repos?: SessionRepo[];
  mainRepoIndex?: number;
  activeWorkflow?: {
    gitUrl: string;
    branch: string;
    path?: string;
  };
};

export type ReconciledRepo = {
  url: string;
  branch: string; // DEPRECATED: Use currentActiveBranch instead
  name?: string;
  branches?: string[]; // All local branches available
  currentActiveBranch?: string; // Currently checked out branch
  defaultBranch?: string; // Default branch of remote
  status?: 'Cloning' | 'Ready' | 'Failed';
  clonedAt?: string;
};

export type ReconciledWorkflow = {
  gitUrl: string;
  branch: string;
  status?: 'Cloning' | 'Active' | 'Failed';
  appliedAt?: string;
};

export type SessionCondition = {
  type: string;
  status: 'True' | 'False' | 'Unknown';
  reason?: string;
  message?: string;
  lastTransitionTime?: string;
  observedGeneration?: number;
};

export type AgenticSessionStatus = {
  observedGeneration?: number;
  phase: AgenticSessionPhase;
  startTime?: string;
  completionTime?: string;
  jobName?: string;
  runnerPodName?: string;
  reconciledRepos?: ReconciledRepo[];
  reconciledWorkflow?: ReconciledWorkflow;
  sdkSessionId?: string;
  sdkRestartCount?: number;
  conditions?: SessionCondition[];
};

export type AgenticSession = {
  metadata: {
    name: string;
    namespace: string;
    creationTimestamp: string;
    uid: string;
    labels?: Record<string, string>;
    annotations?: Record<string, string>;
  };
  spec: AgenticSessionSpec;
  status?: AgenticSessionStatus;
  // Computed field from backend - auto-generated branch name
  // IMPORTANT: Keep in sync with backend (sessions.go) and runner (main.py)
  autoBranch?: string;
};

export type CreateAgenticSessionRequest = {
  initialPrompt?: string;
  llmSettings?: Partial<LLMSettings>;
  displayName?: string;
  timeout?: number;
  project?: string;
  parent_session_id?: string;
  environmentVariables?: Record<string, string>;
  interactive?: boolean;
  repos?: SessionRepo[];
  userContext?: UserContext;
  labels?: Record<string, string>;
  annotations?: Record<string, string>;
};

export type CreateAgenticSessionResponse = {
  message: string;
  name: string;
  uid: string;
  autoBranch: string;  // Auto-generated branch name (e.g., "ambient/1234567890")
};

export type GetAgenticSessionResponse = {
  session: AgenticSession;
};

/**
 * Legacy response type (deprecated - use PaginatedResponse<AgenticSession>)
 */
export type ListAgenticSessionsResponse = {
  items: AgenticSession[];
};

/**
 * Paginated sessions response from the backend
 */
export type ListAgenticSessionsPaginatedResponse = {
  items: AgenticSession[];
  totalCount: number;
  limit: number;
  offset: number;
  hasMore: boolean;
  nextOffset?: number;
};

export type StopAgenticSessionRequest = {
  reason?: string;
};

export type StopAgenticSessionResponse = {
  message: string;
};

export type CloneAgenticSessionRequest = {
  targetProject: string;
  newSessionName: string;
};

export type CloneAgenticSessionResponse = {
  session: AgenticSession;
};

