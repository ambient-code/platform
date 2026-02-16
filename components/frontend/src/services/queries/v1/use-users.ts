import { useMutation, useQuery, useQueryClient, keepPreviousData } from '@tanstack/react-query';
import type { UserCreateRequest, UserPatchRequest, ListOptions } from '@ambient-platform/sdk';
import { createAmbientClient } from '@/lib/ambient-client';
import { userKeys } from './keys';

export function useV1Users(project: string, opts?: ListOptions) {
  return useQuery({
    queryKey: userKeys.list(opts),
    queryFn: () => createAmbientClient(project).users.list(opts),
    enabled: !!project,
    placeholderData: keepPreviousData,
  });
}

export function useV1User(project: string, id: string) {
  return useQuery({
    queryKey: userKeys.detail(id),
    queryFn: () => createAmbientClient(project).users.get(id),
    enabled: !!project && !!id,
  });
}

export function useV1CreateUser(project: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: UserCreateRequest) =>
      createAmbientClient(project).users.create(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: userKeys.lists() });
    },
  });
}

export function useV1UpdateUser(project: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: UserPatchRequest }) =>
      createAmbientClient(project).users.update(id, data),
    onSuccess: (_result, { id }) => {
      queryClient.invalidateQueries({ queryKey: userKeys.detail(id) });
      queryClient.invalidateQueries({ queryKey: userKeys.lists() });
    },
  });
}

export function useV1DeleteUser(project: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) =>
      createAmbientClient(project).users.delete(id),
    onSuccess: (_result, id) => {
      queryClient.removeQueries({ queryKey: userKeys.detail(id) });
      queryClient.invalidateQueries({ queryKey: userKeys.lists() });
    },
  });
}
