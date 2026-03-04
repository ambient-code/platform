import { useMutation, useQuery, useQueryClient, keepPreviousData } from '@tanstack/react-query';
import type { PermissionCreateRequest, PermissionPatchRequest, ListOptions } from '@ambient-platform/sdk';
import { createAmbientClient } from '@/lib/ambient-client';
import { permissionKeys } from './keys';

export function useV1Permissions(project: string, opts?: ListOptions) {
  return useQuery({
    queryKey: permissionKeys.list(opts),
    queryFn: () => createAmbientClient(project).permissions.list(opts),
    enabled: !!project,
    placeholderData: keepPreviousData,
  });
}

export function useV1Permission(project: string, id: string) {
  return useQuery({
    queryKey: permissionKeys.detail(id),
    queryFn: () => createAmbientClient(project).permissions.get(id),
    enabled: !!project && !!id,
  });
}

export function useV1CreatePermission(project: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: PermissionCreateRequest) =>
      createAmbientClient(project).permissions.create(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: permissionKeys.lists() });
    },
  });
}

export function useV1UpdatePermission(project: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: PermissionPatchRequest }) =>
      createAmbientClient(project).permissions.update(id, data),
    onSuccess: (_result, { id }) => {
      queryClient.invalidateQueries({ queryKey: permissionKeys.detail(id) });
      queryClient.invalidateQueries({ queryKey: permissionKeys.lists() });
    },
  });
}
