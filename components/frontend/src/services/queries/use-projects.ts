/**
 * React Query hooks for projects
 */

import { useMutation, useQuery, useQueryClient, keepPreviousData } from '@tanstack/react-query';
import * as projectsApi from '../api/projects';
import type {
  Project,
  CreateProjectRequest,
  UpdateProjectRequest,
  PermissionAssignment,
  PaginationParams,
} from '@/types/api';

/**
 * Query keys for projects
 */
export const projectKeys = {
  all: ['projects'] as const,
  lists: () => [...projectKeys.all, 'list'] as const,
  list: (params?: PaginationParams) => [...projectKeys.lists(), params ?? {}] as const,
  details: () => [...projectKeys.all, 'detail'] as const,
  detail: (name: string) => [...projectKeys.details(), name] as const,
  permissions: (name: string) => [...projectKeys.detail(name), 'permissions'] as const,
  integrationStatus: (name: string) => [...projectKeys.detail(name), 'integration-status'] as const,
};

/**
 * Hook to fetch projects with pagination support
 */
export function useProjectsPaginated(params: PaginationParams = {}) {
  return useQuery({
    queryKey: projectKeys.list(params),
    queryFn: () => projectsApi.listProjectsPaginated(params),
    placeholderData: keepPreviousData, // Keep previous data while fetching new page
  });
}

/**
 * Hook to fetch all projects (legacy - no pagination)
 * @deprecated Use useProjectsPaginated for better performance
 */
export function useProjects() {
  return useQuery({
    queryKey: projectKeys.list(),
    queryFn: projectsApi.listProjects,
  });
}

/**
 * Hook to fetch a single project
 */
export function useProject(name: string) {
  return useQuery({
    queryKey: projectKeys.detail(name),
    queryFn: () => projectsApi.getProject(name),
    enabled: !!name,
  });
}

/**
 * Hook to create a project
 */
export function useCreateProject() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: CreateProjectRequest) => projectsApi.createProject(data),
    onSuccess: () => {
      // Invalidate projects list to refetch
      queryClient.invalidateQueries({ queryKey: projectKeys.lists() });
    },
  });
}

/**
 * Hook to update a project
 */
export function useUpdateProject() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      name,
      data,
    }: {
      name: string;
      data: UpdateProjectRequest;
    }) => projectsApi.updateProject(name, data),
    onSuccess: (project: Project) => {
      // Update cached project details
      queryClient.setQueryData(projectKeys.detail(project.name), project);
      // Invalidate lists to reflect changes
      queryClient.invalidateQueries({ queryKey: projectKeys.lists() });
    },
  });
}

/**
 * Hook to delete a project
 *
 * Implements optimistic updates to prevent race conditions when deleting
 * multiple projects in quick succession.
 */
export function useDeleteProject() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (name: string) => projectsApi.deleteProject(name),
    onMutate: async (name) => {
      // Cancel any outgoing refetches (so they don't overwrite our optimistic update)
      await queryClient.cancelQueries({ queryKey: projectKeys.lists() });

      // Snapshot all list queries for rollback on error
      const previousQueries = new Map<string, unknown>();
      const queries = queryClient.getQueriesData({ queryKey: projectKeys.lists() });
      queries.forEach(([queryKey, data]) => {
        previousQueries.set(JSON.stringify(queryKey), data);
      });

      // Optimistically remove the project from all list queries
      queryClient.setQueriesData(
        { queryKey: projectKeys.lists() },
        (old: unknown) => {
          // Handle paginated response
          if (old && typeof old === 'object' && 'items' in old) {
            const paginatedData = old as { items: Project[]; totalCount?: number };
            return {
              ...paginatedData,
              items: paginatedData.items.filter((p) => p.name !== name),
              totalCount: paginatedData.totalCount ? paginatedData.totalCount - 1 : undefined,
            };
          }
          // Handle legacy array response
          if (Array.isArray(old)) {
            return old.filter((p: Project) => p.name !== name);
          }
          return old;
        }
      );

      // Return context with the snapshots
      return { previousQueries };
    },
    onError: (err, name, context) => {
      // Check if this is a "not found" error (which is fine during deletion)
      const errorMessage = err instanceof Error ? err.message : String(err);
      const isNotFoundError =
        errorMessage.toLowerCase().includes('not found') ||
        errorMessage.includes('404');

      // Only rollback if it's NOT a "not found" error
      if (!isNotFoundError && context?.previousQueries) {
        // Restore all previous query states
        context.previousQueries.forEach((data, keyString) => {
          const queryKey = JSON.parse(keyString);
          queryClient.setQueryData(queryKey, data);
        });
      }
      // Silently ignore "not found" errors during deletion - the project is already gone
    },
    onSuccess: (_data, name) => {
      // Remove the detailed project query from cache
      queryClient.removeQueries({ queryKey: projectKeys.detail(name) });
      // No need to invalidate lists - already optimistically updated
    },
  });
}

/**
 * Hook to fetch project permissions
 */
export function useProjectPermissions(projectName: string) {
  return useQuery({
    queryKey: projectKeys.permissions(projectName),
    queryFn: () => projectsApi.getProjectPermissions(projectName),
    enabled: !!projectName,
  });
}

/**
 * Hook to add project permission
 */
export function useAddProjectPermission() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      projectName,
      permission,
    }: {
      projectName: string;
      permission: PermissionAssignment;
    }) => projectsApi.addProjectPermission(projectName, permission),
    onSuccess: (_data, { projectName }) => {
      // Invalidate permissions to refetch
      queryClient.invalidateQueries({
        queryKey: projectKeys.permissions(projectName),
      });
    },
  });
}

/**
 * Hook to remove project permission
 */
export function useRemoveProjectPermission() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      projectName,
      subjectType,
      subjectName,
    }: {
      projectName: string;
      subjectType: string;
      subjectName: string;
    }) =>
      projectsApi.removeProjectPermission(projectName, subjectType, subjectName),
    onSuccess: (_data, { projectName }) => {
      // Invalidate permissions to refetch
      queryClient.invalidateQueries({
        queryKey: projectKeys.permissions(projectName),
      });
    },
  });
}

/**
 * Hook to fetch project integration status (GitHub, etc.)
 */
export function useProjectIntegrationStatus(projectName: string) {
  return useQuery({
    queryKey: projectKeys.integrationStatus(projectName),
    queryFn: () => projectsApi.getProjectIntegrationStatus(projectName),
    enabled: !!projectName,
    staleTime: 60000, // Cache for 1 minute
  });
}
