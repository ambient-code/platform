/**
 * Google Drive integration API client with TanStack React Query v5 hooks.
 *
 * All endpoints are scoped under:
 *   /api/projects/{projectName}/integrations/google-drive
 */

import {
  useQuery,
  useMutation,
  useQueryClient,
} from "@tanstack/react-query";

// ---------------------------------------------------------------------------
// Type definitions
// ---------------------------------------------------------------------------

export type PermissionScope = "granular" | "full";

export type IntegrationStatus =
  | "active"
  | "disconnected"
  | "expired"
  | "error";

export type FileGrantStatus = "active" | "unavailable" | "revoked";

/** Represents a project-level Google Drive integration. */
export interface DriveIntegration {
  id: string;
  projectName: string;
  permissionScope: PermissionScope;
  status: IntegrationStatus;
  fileCount: number;
  createdAt: string;
  updatedAt: string;
}

/** A file (or folder) that has been granted access within an integration. */
export interface FileGrant {
  id: string;
  googleFileId: string;
  fileName: string;
  mimeType: string;
  fileUrl: string;
  sizeBytes: number | null;
  isFolder: boolean;
  status: FileGrantStatus;
  grantedAt: string;
}

/** Lightweight representation of a file selected via the Google Picker UI. */
export interface PickerFile {
  id: string;
  name: string;
  mimeType: string;
  url: string;
  sizeBytes: number | null;
  isFolder: boolean;
}

// -- Request types ----------------------------------------------------------

export interface InitDriveSetupRequest {
  projectName: string;
  permissionScope: PermissionScope;
  redirectUri: string;
}

export interface HandleDriveCallbackRequest {
  projectName: string;
  code: string;
  state: string;
}

export interface UpdateFileGrantsRequest {
  projectName: string;
  files: PickerFile[];
}

// -- Response types ---------------------------------------------------------

export interface InitDriveSetupResponse {
  authUrl: string;
  state: string;
}

export interface HandleDriveCallbackResponse {
  integration: DriveIntegration;
}

export interface PickerTokenResponse {
  accessToken: string;
  expiresIn: number;
}

export interface ListFileGrantsResponse {
  files: FileGrant[];
}

export interface UpdateFileGrantsResponse {
  files: FileGrant[];
}

export interface DisconnectDriveResponse {
  success: boolean;
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function baseUrl(projectName: string): string {
  return `/api/projects/${encodeURIComponent(projectName)}/integrations/google-drive`;
}

class ApiError extends Error {
  constructor(
    message: string,
    public readonly status: number,
    public readonly body?: unknown,
  ) {
    super(message);
    this.name = "ApiError";
  }
}

async function handleResponse<T>(response: Response): Promise<T> {
  if (!response.ok) {
    let body: unknown;
    try {
      body = await response.json();
    } catch {
      // Response body may not be JSON; that is fine.
    }

    const message =
      typeof body === "object" && body !== null && "message" in body
        ? String((body as Record<string, unknown>).message)
        : `Request failed with status ${response.status}`;

    throw new ApiError(message, response.status, body);
  }

  return response.json() as Promise<T>;
}

// ---------------------------------------------------------------------------
// API functions
// ---------------------------------------------------------------------------

/**
 * Initiate the OAuth setup flow for Google Drive.
 * Returns an authorization URL the frontend should redirect to.
 */
export async function initDriveSetup(
  projectName: string,
  permissionScope: PermissionScope,
  redirectUri: string,
): Promise<InitDriveSetupResponse> {
  const response = await fetch(`${baseUrl(projectName)}/setup`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ permissionScope, redirectUri }),
  });

  return handleResponse<InitDriveSetupResponse>(response);
}

/**
 * Exchange the OAuth callback code for tokens and finalise integration setup.
 */
export async function handleDriveCallback(
  projectName: string,
  code: string,
  state: string,
): Promise<HandleDriveCallbackResponse> {
  const params = new URLSearchParams({ code, state });
  const response = await fetch(
    `${baseUrl(projectName)}/callback?${params.toString()}`,
    { method: "GET" },
  );

  return handleResponse<HandleDriveCallbackResponse>(response);
}

/**
 * Obtain a short-lived access token for the Google Picker UI.
 */
export async function getPickerToken(
  projectName: string,
): Promise<PickerTokenResponse> {
  const response = await fetch(`${baseUrl(projectName)}/picker-token`, {
    method: "GET",
  });

  return handleResponse<PickerTokenResponse>(response);
}

/**
 * List all file grants for the project's Drive integration.
 */
export async function listFileGrants(
  projectName: string,
): Promise<ListFileGrantsResponse> {
  const response = await fetch(`${baseUrl(projectName)}/files`, {
    method: "GET",
  });

  return handleResponse<ListFileGrantsResponse>(response);
}

/**
 * Replace the set of granted files with the provided list.
 */
export async function updateFileGrants(
  projectName: string,
  files: PickerFile[],
): Promise<UpdateFileGrantsResponse> {
  const response = await fetch(`${baseUrl(projectName)}/files`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ files }),
  });

  return handleResponse<UpdateFileGrantsResponse>(response);
}

/**
 * Get the current Drive integration for a project.
 */
export async function getDriveIntegration(
  projectName: string,
): Promise<DriveIntegration> {
  const response = await fetch(baseUrl(projectName), {
    method: "GET",
  });

  return handleResponse<DriveIntegration>(response);
}

/**
 * Disconnect (delete) the Drive integration for a project.
 */
export async function disconnectDriveIntegration(
  projectName: string,
): Promise<DisconnectDriveResponse> {
  const response = await fetch(baseUrl(projectName), {
    method: "DELETE",
  });

  return handleResponse<DisconnectDriveResponse>(response);
}

// ---------------------------------------------------------------------------
// Query key factory
// ---------------------------------------------------------------------------

export const driveQueryKeys = {
  all: ["drive-integration"] as const,
  integration: (projectName: string) =>
    [...driveQueryKeys.all, "integration", projectName] as const,
  fileGrants: (projectName: string) =>
    [...driveQueryKeys.all, "file-grants", projectName] as const,
  pickerToken: (projectName: string) =>
    [...driveQueryKeys.all, "picker-token", projectName] as const,
} as const;

// ---------------------------------------------------------------------------
// React Query hooks
// ---------------------------------------------------------------------------

/**
 * Initiate the Drive OAuth setup flow.
 *
 * Usage:
 * ```ts
 * const { mutate } = useInitDriveSetup();
 * mutate({ projectName, permissionScope: "readonly", redirectUri: "..." });
 * ```
 */
export function useInitDriveSetup() {
  return useMutation<InitDriveSetupResponse, ApiError, InitDriveSetupRequest>({
    mutationFn: ({ projectName, permissionScope, redirectUri }) =>
      initDriveSetup(projectName, permissionScope, redirectUri),
  });
}

/**
 * Handle the OAuth callback after the user authorises access.
 */
export function useDriveCallback() {
  const queryClient = useQueryClient();

  return useMutation<
    HandleDriveCallbackResponse,
    ApiError,
    HandleDriveCallbackRequest
  >({
    mutationFn: ({ projectName, code, state }) =>
      handleDriveCallback(projectName, code, state),
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({
        queryKey: driveQueryKeys.integration(variables.projectName),
      });
    },
  });
}

/**
 * Fetch a short-lived picker token. Disabled by default; set `enabled` to
 * true when the picker UI is about to be shown.
 *
 * @param projectName - The project to fetch a token for.
 * @param options.enabled - Whether the query should execute (default: false).
 */
export function usePickerToken(
  projectName: string,
  options?: { enabled?: boolean },
) {
  return useQuery<PickerTokenResponse, ApiError>({
    queryKey: driveQueryKeys.pickerToken(projectName),
    queryFn: () => getPickerToken(projectName),
    enabled: options?.enabled ?? false,
    // Picker tokens are short-lived; avoid caching stale values.
    staleTime: 0,
    gcTime: 0,
  });
}

/**
 * List file grants for a project's Drive integration.
 */
export function useFileGrants(projectName: string) {
  return useQuery<ListFileGrantsResponse, ApiError>({
    queryKey: driveQueryKeys.fileGrants(projectName),
    queryFn: () => listFileGrants(projectName),
    enabled: !!projectName,
  });
}

/**
 * Update the set of granted files. Automatically invalidates the file grants
 * query on success so the UI stays in sync.
 */
export function useUpdateFileGrants() {
  const queryClient = useQueryClient();

  return useMutation<
    UpdateFileGrantsResponse,
    ApiError,
    UpdateFileGrantsRequest
  >({
    mutationFn: ({ projectName, files }) =>
      updateFileGrants(projectName, files),
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({
        queryKey: driveQueryKeys.fileGrants(variables.projectName),
      });
    },
  });
}

/**
 * Fetch the Drive integration status for a project.
 */
export function useDriveIntegration(projectName: string) {
  return useQuery<DriveIntegration, ApiError>({
    queryKey: driveQueryKeys.integration(projectName),
    queryFn: () => getDriveIntegration(projectName),
    enabled: !!projectName,
  });
}

/**
 * Disconnect the Drive integration. Invalidates the integration query on
 * success and removes cached file grants.
 */
export function useDisconnectDriveIntegration() {
  const queryClient = useQueryClient();

  return useMutation<
    DisconnectDriveResponse,
    ApiError,
    { projectName: string }
  >({
    mutationFn: ({ projectName }) => disconnectDriveIntegration(projectName),
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({
        queryKey: driveQueryKeys.integration(variables.projectName),
      });
      queryClient.removeQueries({
        queryKey: driveQueryKeys.fileGrants(variables.projectName),
      });
      queryClient.removeQueries({
        queryKey: driveQueryKeys.pickerToken(variables.projectName),
      });
    },
  });
}
