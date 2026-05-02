import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { keysAdapter } from '../adapters/keys';
import type { KeysPort } from '../ports/keys';
import type { CreateKeyRequest } from '../ports/types';
import { BACKEND_VERSION } from './query-keys';

export const keysKeys = {
  all: [BACKEND_VERSION, 'keys'] as const,
  lists: () => [...keysKeys.all, 'list'] as const,
  list: (projectName: string) => [...keysKeys.lists(), projectName] as const,
};

export function useKeys(projectName: string, port: KeysPort = keysAdapter) {
  return useQuery({
    queryKey: keysKeys.list(projectName),
    queryFn: () => port.listKeys(projectName),
    staleTime: 5 * 60 * 1000,
    enabled: !!projectName,
  });
}

export function useCreateKey(port: KeysPort = keysAdapter) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ projectName, data }: { projectName: string; data: CreateKeyRequest }) =>
      port.createKey(projectName, data),
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({ queryKey: keysKeys.list(variables.projectName) });
    },
  });
}

export function useDeleteKey(port: KeysPort = keysAdapter) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ projectName, keyId }: { projectName: string; keyId: string }) =>
      port.deleteKey(projectName, keyId),
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({ queryKey: keysKeys.list(variables.projectName) });
    },
  });
}
