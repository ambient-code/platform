/**
 * Projects API service
 * Handles all project-related API calls
 */

import { apiClient } from './client';
import type {
  Project,
  CreateProjectRequest,
  UpdateProjectRequest,
  ListProjectsPaginatedResponse,
  DeleteProjectResponse,
  PermissionAssignment,
  PaginationParams,
} from '@/types/api';

export type IntegrationStatus = {
  github: boolean;
};

/**
 * List projects with pagination support
 */
export async function listProjectsPaginated(
  params: PaginationParams = {}
): Promise<ListProjectsPaginatedResponse> {
  const searchParams = new URLSearchParams();
  if (params.limit) searchParams.set('limit', params.limit.toString());
  if (params.offset) searchParams.set('offset', params.offset.toString());
  if (params.search) searchParams.set('search', params.search);

  const queryString = searchParams.toString();
  const url = queryString ? `/projects?${queryString}` : '/projects';

  return apiClient.get<ListProjectsPaginatedResponse>(url);
}

/**
 * List all projects (legacy - fetches all without pagination)
 * @deprecated Use listProjectsPaginated for better performance
 */
export async function listProjects(): Promise<Project[]> {
  // For backward compatibility, fetch with a high limit
  const response = await listProjectsPaginated({ limit: 100 });
  return response.items;
}

/**
 * Get a single project by name
 */
export async function getProject(name: string): Promise<Project> {
  return apiClient.get<Project>(`/projects/${name}`);
}

/**
 * Create a new project
 */
export async function createProject(data: CreateProjectRequest): Promise<Project> {
  return apiClient.post<Project, CreateProjectRequest>(
    '/projects',
    data
  );
}

/**
 * Update an existing project
 */
export async function updateProject(
  name: string,
  data: UpdateProjectRequest
): Promise<Project> {
  return apiClient.put<Project, UpdateProjectRequest>(
    `/projects/${name}`,
    data
  );
}

/**
 * Delete a project
 */
export async function deleteProject(name: string): Promise<string> {
  const response = await apiClient.delete<DeleteProjectResponse>(`/projects/${name}`);
  return response.message;
}

/**
 * Get project permissions
 */
export async function getProjectPermissions(
  projectName: string
): Promise<PermissionAssignment[]> {
  const response = await apiClient.get<{ items: PermissionAssignment[] }>(
    `/projects/${projectName}/permissions`
  );
  return response.items;
}

/**
 * Add permission to project
 */
export async function addProjectPermission(
  projectName: string,
  permission: PermissionAssignment
): Promise<PermissionAssignment> {
  return apiClient.post<PermissionAssignment, PermissionAssignment>(
    `/projects/${projectName}/permissions`,
    permission
  );
}

/**
 * Remove permission from project
 */
export async function removeProjectPermission(
  projectName: string,
  subjectType: string,
  subjectName: string
): Promise<void> {
  await apiClient.delete(
    `/projects/${projectName}/permissions/${subjectType}/${subjectName}`
  );
}

/**
 * Get project integration status (GitHub, etc.)
 */
export async function getProjectIntegrationStatus(
  projectName: string
): Promise<IntegrationStatus> {
  return apiClient.get<IntegrationStatus>(
    `/projects/${projectName}/integration-status`
  );
}
