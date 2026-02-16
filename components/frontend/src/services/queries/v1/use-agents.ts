import { useMutation, useQuery, useQueryClient, keepPreviousData } from '@tanstack/react-query';
import type { AgentCreateRequest, AgentPatchRequest, ListOptions } from '@ambient-platform/sdk';
import { createAmbientClient } from '@/lib/ambient-client';
import { agentKeys } from './keys';

export function useV1Agents(project: string, opts?: ListOptions) {
  return useQuery({
    queryKey: agentKeys.list(opts),
    queryFn: () => createAmbientClient(project).agents.list(opts),
    enabled: !!project,
    placeholderData: keepPreviousData,
  });
}

export function useV1Agent(project: string, id: string) {
  return useQuery({
    queryKey: agentKeys.detail(id),
    queryFn: () => createAmbientClient(project).agents.get(id),
    enabled: !!project && !!id,
  });
}

export function useV1CreateAgent(project: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: AgentCreateRequest) =>
      createAmbientClient(project).agents.create(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: agentKeys.lists() });
    },
  });
}

export function useV1UpdateAgent(project: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: AgentPatchRequest }) =>
      createAmbientClient(project).agents.update(id, data),
    onSuccess: (_result, { id }) => {
      queryClient.invalidateQueries({ queryKey: agentKeys.detail(id) });
      queryClient.invalidateQueries({ queryKey: agentKeys.lists() });
    },
  });
}
