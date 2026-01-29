/**
 * React Query hooks for agentic sessions
 */

import { useMutation, useQuery, useQueryClient, keepPreviousData } from '@tanstack/react-query';
import * as sessionsApi from '../api/sessions';
import type {
  AgenticSession,
  CreateAgenticSessionRequest,
  StopAgenticSessionRequest,
  CloneAgenticSessionRequest,
  PaginationParams,
} from '@/types/api';

/**
 * Query keys for sessions
 */
export const sessionKeys = {
  all: ['sessions'] as const,
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

/**
 * Hook to fetch sessions for a project with pagination support
 */
export function useSessionsPaginated(projectName: string, params: PaginationParams = {}) {
  return useQuery({
    queryKey: sessionKeys.list(projectName, params),
    queryFn: () => sessionsApi.listSessionsPaginated(projectName, params),
    enabled: !!projectName,
    placeholderData: keepPreviousData, // Keep previous data while fetching new page
  });
}

/**
 * Hook to fetch sessions for a project (legacy - no pagination)
 * @deprecated Use useSessionsPaginated for better performance
 */
export function useSessions(projectName: string) {
  return useQuery({
    queryKey: sessionKeys.list(projectName),
    queryFn: () => sessionsApi.listSessions(projectName),
    enabled: !!projectName,
  });
}

/**
 * Hook to fetch a single session
 */
export function useSession(projectName: string, sessionName: string) {
  return useQuery({
    queryKey: sessionKeys.detail(projectName, sessionName),
    queryFn: () => sessionsApi.getSession(projectName, sessionName),
    enabled: !!projectName && !!sessionName,
    retry: 3, // Retry failed requests (useful during backend rollouts)
    retryDelay: (attemptIndex) => Math.min(1000 * 2 ** attemptIndex, 10000), // Exponential backoff
    // Poll for status updates based on session phase
    refetchInterval: (query) => {
      const session = query.state.data as AgenticSession | undefined;
      const phase = session?.status?.phase;
      const annotations = session?.metadata?.annotations || {};
      
      // Check if a state transition is pending (user requested start/stop)
      // This catches the case where the phase hasn't updated yet but we know
      // a transition is coming
      const desiredPhase = annotations['ambient-code.io/desired-phase'];
      if (desiredPhase) {
        // Pending transition - poll very aggressively (every 500ms)
        return 500;
      }
      
      // Transitional states - poll aggressively (every 1 second)
      const isTransitioning =
        phase === 'Stopping' ||
        phase === 'Pending' ||
        phase === 'Creating';
      if (isTransitioning) return 1000;
      
      // Running state - poll normally (every 5 seconds)
      if (phase === 'Running') return 5000;
      
      // Terminal states (Stopped, Completed, Failed) - no polling
      return false;
    },
  });
}

// useSessionMessages removed - replaced by AG-UI protocol (useAGUIStream)

/**
 * Hook to create a session
 */
export function useCreateSession() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      projectName,
      data,
    }: {
      projectName: string;
      data: CreateAgenticSessionRequest;
    }) => sessionsApi.createSession(projectName, data),
    onSuccess: (_session, { projectName }) => {
      // Invalidate and refetch sessions list
      queryClient.invalidateQueries({
        queryKey: sessionKeys.list(projectName),
        refetchType: 'all', // Refetch both active and inactive queries
      });
    },
  });
}

/**
 * Hook to stop a session
 */
export function useStopSession() {
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
    }) => sessionsApi.stopSession(projectName, sessionName, data),
    onSuccess: (_message, { projectName, sessionName }) => {
      // Invalidate session details to refetch status
      queryClient.invalidateQueries({
        queryKey: sessionKeys.detail(projectName, sessionName),
        refetchType: 'all',
      });
      // Invalidate list to update session count
      queryClient.invalidateQueries({
        queryKey: sessionKeys.list(projectName),
        refetchType: 'all',
      });
    },
  });
}

/**
 * Hook to start/restart a session
 */
export function useStartSession() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      projectName,
      sessionName,
    }: {
      projectName: string;
      sessionName: string;
    }) => sessionsApi.startSession(projectName, sessionName),
    onSuccess: (_response, { projectName, sessionName }) => {
      // Invalidate session details to refetch status
      queryClient.invalidateQueries({
        queryKey: sessionKeys.detail(projectName, sessionName),
        refetchType: 'all',
      });
      // Invalidate list to update session count
      queryClient.invalidateQueries({
        queryKey: sessionKeys.list(projectName),
        refetchType: 'all',
      });
    },
  });
}

/**
 * Hook to clone a session
 */
export function useCloneSession() {
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
    }) => sessionsApi.cloneSession(projectName, sessionName, data),
    onSuccess: (_session, { projectName }) => {
      // Invalidate and refetch sessions list to show new cloned session
      queryClient.invalidateQueries({
        queryKey: sessionKeys.list(projectName),
        refetchType: 'all', // Refetch both active and inactive queries
      });
    },
  });
}

/**
 * Hook to delete a session
 */
export function useDeleteSession() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      projectName,
      sessionName,
    }: {
      projectName: string;
      sessionName: string;
    }) => sessionsApi.deleteSession(projectName, sessionName),
    onSuccess: (_data, { projectName, sessionName }) => {
      // Remove from cache
      queryClient.removeQueries({
        queryKey: sessionKeys.detail(projectName, sessionName),
      });
      // Invalidate list
      queryClient.invalidateQueries({
        queryKey: sessionKeys.list(projectName),
        refetchType: 'all',
      });
    },
  });
}

// useSendChatMessage and useSendControlMessage removed - replaced by AG-UI protocol

/**
 * Hook to fetch K8s resources (job, pods, PVC) for a session
 */
export function useSessionK8sResources(projectName: string, sessionName: string) {
  return useQuery({
    queryKey: [...sessionKeys.detail(projectName, sessionName), 'k8s-resources'] as const,
    queryFn: () => sessionsApi.getSessionK8sResources(projectName, sessionName),
    enabled: !!projectName && !!sessionName,
    refetchInterval: 5000, // Poll every 5 seconds
  });
}

/**
 * Hook to continue a session (restarts the existing session)
 */
export function useContinueSession() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      projectName,
      parentSessionName,
    }: {
      projectName: string;
      parentSessionName: string;
    }) => {
      // Restart the existing session by updating its status to Creating
      return sessionsApi.startSession(projectName, parentSessionName);
    },
    onSuccess: (_response, { projectName, parentSessionName }) => {
      // Invalidate session details to refetch status
      queryClient.invalidateQueries({
        queryKey: sessionKeys.detail(projectName, parentSessionName),
        refetchType: 'all',
      });
      // Invalidate list to update session count
      queryClient.invalidateQueries({
        queryKey: sessionKeys.list(projectName),
        refetchType: 'all',
      });
    },
  });
}

/**
 * Hook to update a session's display name
 */
export function useUpdateSessionDisplayName() {
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
    }) => sessionsApi.updateSessionDisplayName(projectName, sessionName, displayName),
    onSuccess: (_data, { projectName, sessionName }) => {
      // Invalidate session details to refetch with new name
      queryClient.invalidateQueries({
        queryKey: sessionKeys.detail(projectName, sessionName),
        refetchType: 'all',
      });
      // Invalidate list to update session name in list view
      queryClient.invalidateQueries({
        queryKey: sessionKeys.list(projectName),
        refetchType: 'all',
      });
    },
  });
}

/**
 * Hook to fetch session export data (AG-UI events + legacy messages)
 */
export function useSessionExport(projectName: string, sessionName: string, enabled: boolean) {
  return useQuery({
    queryKey: sessionKeys.export(projectName, sessionName),
    queryFn: () => sessionsApi.getSessionExport(projectName, sessionName),
    enabled: enabled && !!projectName && !!sessionName,
    staleTime: 60000, // Cache for 1 minute
  });
}

/**
 * Hook to fetch repository status (branches, current branch) from runner
 * Polls every 30 seconds for real-time updates
 */
export function useReposStatus(projectName: string, sessionName: string, enabled: boolean = true) {
  return useQuery({
    queryKey: sessionKeys.reposStatus(projectName, sessionName),
    queryFn: () => sessionsApi.getReposStatus(projectName, sessionName),
    enabled: enabled && !!projectName && !!sessionName,
    refetchInterval: 30000, // Poll every 30 seconds
    staleTime: 25000, // Consider stale after 25 seconds
  });
}
