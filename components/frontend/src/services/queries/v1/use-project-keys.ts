import { useMutation, useQuery, useQueryClient, keepPreviousData } from '@tanstack/react-query';
import type { ProjectKeyCreateRequest, ListOptions } from '@ambient-platform/sdk';
import { createAmbientClient } from '@/lib/ambient-client';
import { projectKeyKeys } from './keys';

export function useV1ProjectKeys(project: string, opts?: ListOptions) {
  return useQuery({
    queryKey: projectKeyKeys.list(opts),
    queryFn: () => createAmbientClient(project).projectKeys.list(opts),
    enabled: !!project,
    placeholderData: keepPreviousData,
  });
}

export function useV1ProjectKey(project: string, id: string) {
  return useQuery({
    queryKey: projectKeyKeys.detail(id),
    queryFn: () => createAmbientClient(project).projectKeys.get(id),
    enabled: !!project && !!id,
  });
}

export function useV1CreateProjectKey(project: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: ProjectKeyCreateRequest) =>
      createAmbientClient(project).projectKeys.create(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: projectKeyKeys.lists() });
    },
  });
}

export function useV1DeleteProjectKey(project: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) =>
      createAmbientClient(project).projectKeys.delete(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: projectKeyKeys.lists() });
    },
  });
}
