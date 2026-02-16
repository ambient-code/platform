import { useMutation, useQuery, useQueryClient, keepPreviousData } from '@tanstack/react-query';
import type { ProjectCreateRequest, ProjectPatchRequest, ListOptions } from '@ambient-platform/sdk';
import { createAmbientClient } from '@/lib/ambient-client';
import { projectKeys } from './keys';

export function useV1Projects(project: string, opts?: ListOptions) {
  return useQuery({
    queryKey: projectKeys.list(opts),
    queryFn: () => createAmbientClient(project).projects.list(opts),
    enabled: !!project,
    placeholderData: keepPreviousData,
  });
}

export function useV1ProjectsAll(opts?: ListOptions, queryOpts?: { enabled?: boolean }) {
  return useQuery({
    queryKey: projectKeys.list(opts),
    queryFn: () => createAmbientClient('').projects.list(opts),
    placeholderData: keepPreviousData,
    enabled: queryOpts?.enabled,
  });
}

export function useV1DeleteProjectGlobal() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) =>
      createAmbientClient('').projects.delete(id),
    onSuccess: (_result, id) => {
      queryClient.removeQueries({ queryKey: projectKeys.detail(id) });
      queryClient.invalidateQueries({ queryKey: projectKeys.lists() });
    },
  });
}

export function useV1Project(project: string, id: string) {
  return useQuery({
    queryKey: projectKeys.detail(id),
    queryFn: () => createAmbientClient(project).projects.get(id),
    enabled: !!project && !!id,
  });
}

export function useV1CreateProject(project: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: ProjectCreateRequest) =>
      createAmbientClient(project).projects.create(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: projectKeys.lists() });
    },
  });
}

export function useV1UpdateProject(project: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: ProjectPatchRequest }) =>
      createAmbientClient(project).projects.update(id, data),
    onSuccess: (_result, { id }) => {
      queryClient.invalidateQueries({ queryKey: projectKeys.detail(id) });
      queryClient.invalidateQueries({ queryKey: projectKeys.lists() });
    },
  });
}

export function useV1DeleteProject(project: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) =>
      createAmbientClient(project).projects.delete(id),
    onSuccess: (_result, id) => {
      queryClient.removeQueries({ queryKey: projectKeys.detail(id) });
      queryClient.invalidateQueries({ queryKey: projectKeys.lists() });
    },
  });
}
