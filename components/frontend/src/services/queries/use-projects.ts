import { useMutation, useQuery, useQueryClient, keepPreviousData } from '@tanstack/react-query';
import { projectsAdapter } from '../adapters/projects';
import { projectAccessAdapter } from '../adapters/project-access';
import type { ProjectsPort } from '../ports/projects';
import type { ProjectAccessPort } from '../ports/project-access';
import type {
  Project,
  CreateProjectRequest,
  UpdateProjectRequest,
  PermissionAssignment,
  PaginationParams,
} from '@/types/api';
import type { MCPServersConfig } from '@/types/agentic-session';
import { BACKEND_VERSION } from './query-keys';

export const projectKeys = {
  all: [BACKEND_VERSION, 'projects'] as const,
  lists: () => [...projectKeys.all, 'list'] as const,
  list: (params?: PaginationParams) => [...projectKeys.lists(), params ?? {}] as const,
  details: () => [...projectKeys.all, 'detail'] as const,
  detail: (name: string) => [...projectKeys.details(), name] as const,
  permissions: (name: string) => [...projectKeys.detail(name), 'permissions'] as const,
  integrationStatus: (name: string) => [...projectKeys.detail(name), 'integration-status'] as const,
  mcpServers: (name: string) => [...projectKeys.detail(name), 'mcp-servers'] as const,
};

export function useProjectsPaginated(params: PaginationParams = {}, port: ProjectsPort = projectsAdapter) {
  return useQuery({
    queryKey: projectKeys.list(params),
    queryFn: () => port.listProjects(params),
    placeholderData: keepPreviousData,
  });
}

/** @deprecated Use useProjectsPaginated for better performance */
export function useProjects(port: ProjectsPort = projectsAdapter) {
  return useQuery({
    queryKey: projectKeys.list(),
    queryFn: async () => {
      const result = await port.listProjects();
      return result.items;
    },
  });
}

export function useProject(name: string, port: ProjectsPort = projectsAdapter) {
  return useQuery({
    queryKey: projectKeys.detail(name),
    queryFn: () => port.getProject(name),
    enabled: !!name,
  });
}

export function useCreateProject(port: ProjectsPort = projectsAdapter) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: CreateProjectRequest) => port.createProject(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: projectKeys.lists() });
    },
  });
}

export function useUpdateProject(port: ProjectsPort = projectsAdapter) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      name,
      data,
    }: {
      name: string;
      data: UpdateProjectRequest;
    }) => port.updateProject(name, data),
    onSuccess: (project: Project) => {
      queryClient.setQueryData(projectKeys.detail(project.name), project);
      queryClient.invalidateQueries({ queryKey: projectKeys.lists() });
    },
  });
}

export function useDeleteProject(port: ProjectsPort = projectsAdapter) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (name: string) => port.deleteProject(name),
    onMutate: async (name) => {
      await queryClient.cancelQueries({ queryKey: projectKeys.lists() });

      const previousQueries = new Map<string, unknown>();
      const queries = queryClient.getQueriesData({ queryKey: projectKeys.lists() });
      queries.forEach(([queryKey, data]) => {
        previousQueries.set(JSON.stringify(queryKey), data);
      });

      queryClient.setQueriesData(
        { queryKey: projectKeys.lists() },
        (old: unknown) => {
          if (old && typeof old === 'object' && 'items' in old) {
            const paginatedData = old as { items: Project[]; totalCount?: number };
            return {
              ...paginatedData,
              items: paginatedData.items.filter((p) => p.name !== name),
              totalCount: paginatedData.totalCount ? paginatedData.totalCount - 1 : undefined,
            };
          }
          if (Array.isArray(old)) {
            return old.filter((p: Project) => p.name !== name);
          }
          return old;
        }
      );

      return { previousQueries };
    },
    onError: (err, _name, context) => {
      const errorMessage = err instanceof Error ? err.message : String(err);
      const isNotFoundError =
        errorMessage.toLowerCase().includes('not found') ||
        errorMessage.includes('404');

      if (!isNotFoundError && context?.previousQueries) {
        context.previousQueries.forEach((data, keyString) => {
          const queryKey = JSON.parse(keyString);
          queryClient.setQueryData(queryKey, data);
        });
      }
    },
    onSuccess: (_data, name) => {
      queryClient.removeQueries({ queryKey: projectKeys.detail(name) });
    },
  });
}

export function useProjectPermissions(projectName: string, port: ProjectAccessPort = projectAccessAdapter) {
  return useQuery({
    queryKey: projectKeys.permissions(projectName),
    queryFn: () => port.getPermissions(projectName),
    enabled: !!projectName,
  });
}

export function useAddProjectPermission(port: ProjectAccessPort = projectAccessAdapter) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      projectName,
      permission,
    }: {
      projectName: string;
      permission: PermissionAssignment;
    }) => port.addPermission(projectName, permission),
    onSuccess: (_data, { projectName }) => {
      queryClient.invalidateQueries({
        queryKey: projectKeys.permissions(projectName),
      });
    },
  });
}

export function useRemoveProjectPermission(port: ProjectAccessPort = projectAccessAdapter) {
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
    }) => port.removePermission(projectName, subjectType, subjectName),
    onSuccess: (_data, { projectName }) => {
      queryClient.invalidateQueries({
        queryKey: projectKeys.permissions(projectName),
      });
    },
  });
}

export function useProjectIntegrationStatus(projectName: string, port: ProjectsPort = projectsAdapter) {
  return useQuery({
    queryKey: projectKeys.integrationStatus(projectName),
    queryFn: () => port.getProjectIntegrationStatus(projectName),
    enabled: !!projectName,
    staleTime: 60000,
    refetchOnMount: 'always',
  });
}

export function useProjectMcpServers(projectName: string, port: ProjectsPort = projectsAdapter) {
  return useQuery({
    queryKey: projectKeys.mcpServers(projectName),
    queryFn: () => port.getProjectMcpServers(projectName),
    enabled: !!projectName,
    staleTime: 30000,
  });
}

export function useUpdateProjectMcpServers(projectName: string, port: ProjectsPort = projectsAdapter) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (config: MCPServersConfig) =>
      port.updateProjectMcpServers(projectName, config),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: projectKeys.mcpServers(projectName),
      });
    },
  });
}
