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
 * Pod event from the runner pod's Kubernetes events
 */
export type PodEvent = {
  type: string;      // "Normal" | "Warning"
  reason: string;    // "Scheduled", "Pulling", "Pulled", "Created", "Started", "FailedScheduling", etc.
  message: string;
  timestamp: string; // RFC3339
  count: number;
};

export type PodEventsResponse = {
  events: PodEvent[];
};

/**
 * Get Kubernetes events for the session's runner pod.
 * Lightweight alternative to the old k8s-resources endpoint.
 */
export async function getSessionPodEvents(
  projectName: string,
  sessionName: string
): Promise<PodEventsResponse> {
  return apiClient.get(`/projects/${projectName}/agentic-sessions/${sessionName}/pod-events`);
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

/**
 * Response from Google Drive file creation
 */
export type GoogleDriveFileResponse = {
  content?: string;
  error?: string;
};

/**
 * Save content to Google Drive via the session's MCP server
 */
export async function saveToGoogleDrive(
  projectName: string,
  sessionName: string,
  content: string,
  filename: string,
  userEmail: string,
  serverName: string = 'google-workspace',
): Promise<GoogleDriveFileResponse> {
  return apiClient.post<GoogleDriveFileResponse>(
    `/projects/${projectName}/agentic-sessions/${sessionName}/mcp/invoke`,
    {
      server: serverName,
      tool: 'create_drive_file',
      args: { user_google_email: userEmail, file_name: filename, content, mime_type: 'text/markdown' },
    },
  );
}

// --- Capabilities ---

export type CapabilitiesResponse = {
  framework: string;
  agent_features: string[];
  platform_features: string[];
  file_system: boolean;
  mcp: boolean;
  tracing: string | null;
  session_persistence: boolean;
  model: string | null;
  session_id: string | null;
};

export async function getCapabilities(
  projectName: string,
  sessionName: string
): Promise<CapabilitiesResponse> {
  return apiClient.get<CapabilitiesResponse>(
    `/projects/${projectName}/agentic-sessions/${sessionName}/agui/capabilities`
  );
}
