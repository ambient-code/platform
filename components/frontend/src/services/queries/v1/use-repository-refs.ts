import { useMutation, useQuery, useQueryClient, keepPreviousData } from '@tanstack/react-query';
import type { RepositoryRefCreateRequest, RepositoryRefPatchRequest, ListOptions } from '@ambient-platform/sdk';
import { createAmbientClient } from '@/lib/ambient-client';
import { repositoryRefKeys } from './keys';

export function useV1RepositoryRefs(project: string, opts?: ListOptions) {
  return useQuery({
    queryKey: repositoryRefKeys.list(opts),
    queryFn: () => createAmbientClient(project).repositoryRefs.list(opts),
    enabled: !!project,
    placeholderData: keepPreviousData,
  });
}

export function useV1RepositoryRef(project: string, id: string) {
  return useQuery({
    queryKey: repositoryRefKeys.detail(id),
    queryFn: () => createAmbientClient(project).repositoryRefs.get(id),
    enabled: !!project && !!id,
  });
}

export function useV1CreateRepositoryRef(project: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: RepositoryRefCreateRequest) =>
      createAmbientClient(project).repositoryRefs.create(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: repositoryRefKeys.lists() });
    },
  });
}

export function useV1UpdateRepositoryRef(project: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: RepositoryRefPatchRequest }) =>
      createAmbientClient(project).repositoryRefs.update(id, data),
    onSuccess: (_result, { id }) => {
      queryClient.invalidateQueries({ queryKey: repositoryRefKeys.detail(id) });
      queryClient.invalidateQueries({ queryKey: repositoryRefKeys.lists() });
    },
  });
}
