import { useMutation, useQuery, useQueryClient, keepPreviousData } from '@tanstack/react-query';
import type {
  Session,
  SessionCreateRequest,
  SessionPatchRequest,
  ListOptions,
} from '@ambient-platform/sdk';
import { createAmbientClient } from '@/lib/ambient-client';
import { sessionKeys } from './keys';

export function useV1Sessions(project: string, opts?: ListOptions) {
  return useQuery({
    queryKey: sessionKeys.list(opts),
    queryFn: () => createAmbientClient(project).sessions.list(opts),
    enabled: !!project,
    placeholderData: keepPreviousData,
  });
}

export function useV1Session(project: string, id: string, options?: { enabled?: boolean }) {
  return useQuery({
    queryKey: sessionKeys.detail(id),
    queryFn: () => createAmbientClient(project).sessions.get(id),
    enabled: (options?.enabled ?? true) && !!project && !!id,
    refetchInterval: (query) => {
      const session = query.state.data as Session | undefined;
      if (!session) return false;
      const phase = session.phase;
      if (phase === 'Stopping' || phase === 'Pending' || phase === 'Creating') return 1000;
      if (phase === 'Running') return 5000;
      return false;
    },
  });
}

export function useV1CreateSession(project: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: SessionCreateRequest) =>
      createAmbientClient(project).sessions.create(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: sessionKeys.lists() });
    },
  });
}

export function useV1UpdateSession(project: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: SessionPatchRequest }) =>
      createAmbientClient(project).sessions.update(id, data),
    onSuccess: (_result, { id }) => {
      queryClient.invalidateQueries({ queryKey: sessionKeys.detail(id) });
      queryClient.invalidateQueries({ queryKey: sessionKeys.lists() });
    },
  });
}

export function useV1StartSession(project: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) =>
      createAmbientClient(project).sessions.start(id),
    onSuccess: (_result, id) => {
      queryClient.invalidateQueries({ queryKey: sessionKeys.detail(id) });
      queryClient.invalidateQueries({ queryKey: sessionKeys.lists() });
    },
  });
}

export function useV1StopSession(project: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) =>
      createAmbientClient(project).sessions.stop(id),
    onSuccess: (_result, id) => {
      queryClient.invalidateQueries({ queryKey: sessionKeys.detail(id) });
      queryClient.invalidateQueries({ queryKey: sessionKeys.lists() });
    },
  });
}
