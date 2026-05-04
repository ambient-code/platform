import { useMutation, useQuery, useQueryClient, keepPreviousData } from '@tanstack/react-query';
import { sessionsAdapter } from '../adapters/sessions';
import { sessionReposAdapter } from '../adapters/session-repos';
import type { SessionsPort } from '../ports/sessions';
import type { SessionReposPort } from '../ports/session-repos';
import type {
  AgenticSession,
  CreateAgenticSessionRequest,
  StopAgenticSessionRequest,
  CloneAgenticSessionRequest,
  PaginationParams,
} from '@/types/api';
import { BACKEND_VERSION } from './query-keys';

export const sessionKeys = {
  all: [BACKEND_VERSION, 'sessions'] as const,
  lists: () => [...sessionKeys.all, 'list'] as const,
  list: (projectName: string, params?: PaginationParams) =>
    [...sessionKeys.lists(), projectName, params ?? {}] as const,
  details: () => [...sessionKeys.all, 'detail'] as const,
  detail: (projectName: string, sessionName: string) =>
    [...sessionKeys.details(), projectName, sessionName] as const,
  messages: (projectName: string, sessionName: string) =>
    [...sessionKeys.detail(projectName, sessionName), 'messages'] as const,
  export: (projectName: string, sessionName: string) =>
    [...sessionKeys.detail(projectName, sessionName), 'export'] as const,
  reposStatus: (projectName: string, sessionName: string) =>
    [...sessionKeys.detail(projectName, sessionName), 'repos-status'] as const,
};

export function useSessionsPaginated(projectName: string, params: PaginationParams = {}, port: SessionsPort = sessionsAdapter) {
  return useQuery({
    queryKey: sessionKeys.list(projectName, params),
    queryFn: () => port.listSessions(projectName, params),
    enabled: !!projectName,
    placeholderData: keepPreviousData,
    refetchOnMount: 'always',
    refetchInterval: (query) => {
      const data = query.state.data as { items?: AgenticSession[] } | undefined;
      const items = data?.items;
      if (!items?.length) return false;

      const hasTransitioning = items.some((s) => {
        const phase = s.status?.phase;
        return phase === 'Pending' || phase === 'Creating' || phase === 'Stopping';
      });
      if (hasTransitioning) return 2000;

      const hasWorking = items.some((s) => {
        return s.status?.phase === 'Running' && (!s.status?.agentStatus || s.status?.agentStatus === 'working');
      });
      if (hasWorking) return 5000;

      const hasRunning = items.some((s) => s.status?.phase === 'Running');
      if (hasRunning) return 15000;

      return false;
    },
  });
}

/** @deprecated Use useSessionsPaginated for better performance */
export function useSessions(projectName: string, port: SessionsPort = sessionsAdapter) {
  return useQuery({
    queryKey: sessionKeys.list(projectName),
    queryFn: async () => {
      const result = await port.listSessions(projectName);
      return result.items;
    },
    enabled: !!projectName,
  });
}

export function useSession(projectName: string, sessionName: string, port: SessionsPort = sessionsAdapter) {
  return useQuery({
    queryKey: sessionKeys.detail(projectName, sessionName),
    queryFn: () => port.getSession(projectName, sessionName),
    enabled: !!projectName && !!sessionName,
    retry: 3,
    retryDelay: (attemptIndex) => Math.min(1000 * 2 ** attemptIndex, 10000),
    refetchInterval: (query) => {
      const session = query.state.data as AgenticSession | undefined;
      const phase = session?.status?.phase;
      const annotations = session?.metadata?.annotations || {};

      const desiredPhase = annotations['ambient-code.io/desired-phase'];
      if (desiredPhase) {
        return 500;
      }

      const isTransitioning =
        phase === 'Stopping' ||
        phase === 'Pending' ||
        phase === 'Creating';
      if (isTransitioning) return 1000;

      if (phase === 'Running') return 5000;

      return false;
    },
  });
}

export function useCreateSession(port: SessionsPort = sessionsAdapter) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      projectName,
      data,
    }: {
      projectName: string;
      data: CreateAgenticSessionRequest;
    }) => port.createSession(projectName, data),
    onSuccess: (_session, { projectName }) => {
      queryClient.invalidateQueries({
        queryKey: sessionKeys.list(projectName),
        refetchType: 'all',
      });
    },
  });
}

export function useStopSession(port: SessionsPort = sessionsAdapter) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      projectName,
      sessionName,
      data,
    }: {
      projectName: string;
      sessionName: string;
      data?: StopAgenticSessionRequest;
    }) => port.stopSession(projectName, sessionName, data),
    onSuccess: (_message, { projectName, sessionName }) => {
      queryClient.invalidateQueries({
        queryKey: sessionKeys.detail(projectName, sessionName),
        refetchType: 'all',
      });
      queryClient.invalidateQueries({
        queryKey: sessionKeys.list(projectName),
        refetchType: 'all',
      });
    },
  });
}

export function useStartSession(port: SessionsPort = sessionsAdapter) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      projectName,
      sessionName,
    }: {
      projectName: string;
      sessionName: string;
    }) => port.startSession(projectName, sessionName),
    onSuccess: (_response, { projectName, sessionName }) => {
      queryClient.invalidateQueries({
        queryKey: sessionKeys.detail(projectName, sessionName),
        refetchType: 'all',
      });
      queryClient.invalidateQueries({
        queryKey: sessionKeys.list(projectName),
        refetchType: 'all',
      });
    },
  });
}

export function useCloneSession(port: SessionsPort = sessionsAdapter) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      projectName,
      sessionName,
      data,
    }: {
      projectName: string;
      sessionName: string;
      data: CloneAgenticSessionRequest;
    }) => port.cloneSession(projectName, sessionName, data),
    onSuccess: (_session, { projectName }) => {
      queryClient.invalidateQueries({
        queryKey: sessionKeys.list(projectName),
        refetchType: 'all',
      });
    },
  });
}

export function useDeleteSession(port: SessionsPort = sessionsAdapter) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      projectName,
      sessionName,
    }: {
      projectName: string;
      sessionName: string;
    }) => port.deleteSession(projectName, sessionName),
    onSuccess: (_data, { projectName, sessionName }) => {
      queryClient.removeQueries({
        queryKey: sessionKeys.detail(projectName, sessionName),
      });
      queryClient.invalidateQueries({
        queryKey: sessionKeys.list(projectName),
        refetchType: 'all',
      });
    },
  });
}

export function useSessionPodEvents(
  projectName: string,
  sessionName: string,
  refetchInterval: number = 3000,
  port: SessionsPort = sessionsAdapter,
) {
  return useQuery({
    queryKey: [...sessionKeys.detail(projectName, sessionName), 'pod-events'] as const,
    queryFn: () => port.getSessionPodEvents(projectName, sessionName),
    enabled: !!projectName && !!sessionName,
    refetchInterval,
  });
}

export function useContinueSession(port: SessionsPort = sessionsAdapter) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      projectName,
      parentSessionName,
    }: {
      projectName: string;
      parentSessionName: string;
    }) => {
      return port.startSession(projectName, parentSessionName);
    },
    onSuccess: (_response, { projectName, parentSessionName }) => {
      queryClient.invalidateQueries({
        queryKey: sessionKeys.detail(projectName, parentSessionName),
        refetchType: 'all',
      });
      queryClient.invalidateQueries({
        queryKey: sessionKeys.list(projectName),
        refetchType: 'all',
      });
    },
  });
}

export function useUpdateSessionDisplayName(port: SessionsPort = sessionsAdapter) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      projectName,
      sessionName,
      displayName,
    }: {
      projectName: string;
      sessionName: string;
      displayName: string;
    }) => port.updateSessionDisplayName(projectName, sessionName, displayName),
    onSuccess: (_data, { projectName, sessionName }) => {
      queryClient.invalidateQueries({
        queryKey: sessionKeys.detail(projectName, sessionName),
        refetchType: 'all',
      });
      queryClient.invalidateQueries({
        queryKey: sessionKeys.list(projectName),
        refetchType: 'all',
      });
    },
  });
}

export function useSwitchSessionModel(port: SessionsPort = sessionsAdapter) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      projectName,
      sessionName,
      model,
    }: {
      projectName: string;
      sessionName: string;
      model: string;
    }) => port.switchSessionModel(projectName, sessionName, model),
    onSuccess: (_data, { projectName, sessionName }) => {
      queryClient.invalidateQueries({
        queryKey: sessionKeys.detail(projectName, sessionName),
        refetchType: 'all',
      });
    },
  });
}

export function useSessionExport(projectName: string, sessionName: string, enabled: boolean, port: SessionsPort = sessionsAdapter) {
  return useQuery({
    queryKey: sessionKeys.export(projectName, sessionName),
    queryFn: () => port.getSessionExport(projectName, sessionName),
    enabled: enabled && !!projectName && !!sessionName,
    staleTime: 60000,
  });
}

export function useReposStatus(projectName: string, sessionName: string, enabled: boolean = true, port: SessionReposPort = sessionReposAdapter) {
  return useQuery({
    queryKey: sessionKeys.reposStatus(projectName, sessionName),
    queryFn: () => port.getReposStatus(projectName, sessionName),
    enabled: enabled && !!projectName && !!sessionName,
    refetchInterval: 30000,
    staleTime: 25000,
  });
}
