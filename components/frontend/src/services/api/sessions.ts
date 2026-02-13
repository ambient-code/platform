/**
 * Agentic Sessions API service
 * Handles all session-related API calls
 */

import { apiClient } from './client';
import type {
  AgenticSession,
  CreateAgenticSessionRequest,
  CreateAgenticSessionResponse,
  GetAgenticSessionResponse,
  ListAgenticSessionsPaginatedResponse,
  StopAgenticSessionRequest,
  StopAgenticSessionResponse,
  CloneAgenticSessionRequest,
  CloneAgenticSessionResponse,
  PaginationParams,
} from '@/types/api';

export type McpToolAnnotations = {
  readOnly?: boolean;
  destructive?: boolean;
  idempotent?: boolean;
  openWorld?: boolean;
  [key: string]: boolean | undefined;
};

export type McpTool = {
  name: string;
  annotations: McpToolAnnotations;
};

export type McpServer = {
  name: string;
  displayName: string;
  status: string;
  version?: string;
  tools?: McpTool[];
};

export type McpStatusResponse = {
  servers: McpServer[];
  totalCount: number;
};

/**
 * List sessions for a project with pagination support
 */
export async function listSessionsPaginated(
  projectName: string,
  params: PaginationParams = {}
): Promise<ListAgenticSessionsPaginatedResponse> {
  const searchParams = new URLSearchParams();
  if (params.limit) searchParams.set('limit', params.limit.toString());
  if (params.offset) searchParams.set('offset', params.offset.toString());
  if (params.search) searchParams.set('search', params.search);

  const queryString = searchParams.toString();
  const url = queryString
    ? `/projects/${projectName}/agentic-sessions?${queryString}`
    : `/projects/${projectName}/agentic-sessions`;

  return apiClient.get<ListAgenticSessionsPaginatedResponse>(url);
}

/**
 * List sessions for a project (legacy - fetches all without pagination)
 * @deprecated Use listSessionsPaginated for better performance
 */
export async function listSessions(projectName: string): Promise<AgenticSession[]> {
  // For backward compatibility, fetch with a high limit
  const response = await listSessionsPaginated(projectName, { limit: 100 });
  return response.items;
}

/**
 * Get a single session
 */
export async function getSession(
  projectName: string,
  sessionName: string
): Promise<AgenticSession> {
  const response = await apiClient.get<GetAgenticSessionResponse | AgenticSession>(
    `/projects/${projectName}/agentic-sessions/${sessionName}`
  );
  // Handle both wrapped and unwrapped responses
  if ('session' in response && response.session) {
    return response.session;
  }
  return response as AgenticSession;
}

/**
 * Create a new session
 */
export async function createSession(
  projectName: string,
  data: CreateAgenticSessionRequest
): Promise<AgenticSession> {
  const response = await apiClient.post<
    CreateAgenticSessionResponse,
    CreateAgenticSessionRequest
  >(`/projects/${projectName}/agentic-sessions`, data);
  
  // Backend returns simplified response, fetch the full session object
  return await getSession(projectName, response.name);
}

/**
 * Stop a running session
 */
export async function stopSession(
  projectName: string,
  sessionName: string,
  data?: StopAgenticSessionRequest
): Promise<string> {
  const response = await apiClient.post<
    StopAgenticSessionResponse,
    StopAgenticSessionRequest | undefined
  >(`/projects/${projectName}/agentic-sessions/${sessionName}/stop`, data);
  return response.message;
}

/**
 * Start/restart a session
 */
export async function startSession(
  projectName: string,
  sessionName: string
): Promise<{ message: string }> {
  return apiClient.post<{ message: string }>(
    `/projects/${projectName}/agentic-sessions/${sessionName}/start`
  );
}

/**
 * Clone an existing session
 */
export async function cloneSession(
  projectName: string,
  sessionName: string,
  data: CloneAgenticSessionRequest
): Promise<AgenticSession> {
  const response = await apiClient.post<
    CloneAgenticSessionResponse,
    CloneAgenticSessionRequest
  >(`/projects/${projectName}/agentic-sessions/${sessionName}/clone`, data);
  return response.session;
}

// getSessionMessages removed - replaced by AG-UI protocol

/**
 * Delete a session
 */
export async function deleteSession(
  projectName: string,
  sessionName: string
): Promise<void> {
  await apiClient.delete(`/projects/${projectName}/agentic-sessions/${sessionName}`);
}

// sendChatMessage and sendControlMessage removed - use AG-UI protocol

/**
 * Get K8s resource information (job, pods, PVC) for a session
 */
export async function getSessionK8sResources(
  projectName: string,
  sessionName: string
): Promise<{
  jobName: string;
  jobStatus?: string;
  pods?: Array<{
    name: string;
    phase: string;
    containers: Array<{
      name: string;
      state: string;
      exitCode?: number;
      reason?: string;
    }>;
  }>;
  pvcName: string;
  pvcExists: boolean;
  pvcSize?: string;
}> {
  return apiClient.get(`/projects/${projectName}/agentic-sessions/${sessionName}/k8s-resources`);
}

/**
 * Update the display name of a session
 */
export async function updateSessionDisplayName(
  projectName: string,
  sessionName: string,
  displayName: string
): Promise<AgenticSession> {
  return apiClient.put<AgenticSession, { displayName: string }>(
    `/projects/${projectName}/agentic-sessions/${sessionName}/displayname`,
    { displayName }
  );
}

/**
 * Export session chat data
 */
export type SessionExportResponse = {
  sessionId: string;
  projectName: string;
  exportDate: string;
  aguiEvents: unknown[];
  legacyMessages?: unknown[];
  hasLegacy: boolean;
};

export async function getSessionExport(
  projectName: string,
  sessionName: string
): Promise<SessionExportResponse> {
  return apiClient.get(`/projects/${projectName}/agentic-sessions/${sessionName}/export`);
}

/**
 * Get MCP server status for a session
 */
export async function getMcpStatus(
  projectName: string,
  sessionName: string
): Promise<McpStatusResponse> {
  return apiClient.get<McpStatusResponse>(
    `/projects/${projectName}/agentic-sessions/${sessionName}/mcp/status`
  );
}

export type RepoStatus = {
  url: string;
  name: string;
  branches: string[];
  currentActiveBranch: string;
  defaultBranch: string;
};

export type ReposStatusResponse = {
  repos: RepoStatus[];
};

/**
 * Get current status of all repositories (branches, current branch, etc.)
 * Fetches directly from runner for real-time updates
 */
export async function getReposStatus(
  projectName: string,
  sessionName: string
): Promise<ReposStatusResponse> {
  return apiClient.get<ReposStatusResponse>(
    `/projects/${projectName}/agentic-sessions/${sessionName}/repos/status`
  );
}
