import { useMemo } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useApiSource } from '@/contexts/api-source-context';
import { useSession } from './use-sessions';
import { useV1Session } from './v1/use-sessions';
import { useStopSession, useDeleteSession, useContinueSession } from './use-sessions';
import { v1SessionToAgenticSession } from '@/lib/v1-session-adapter';
import { createAmbientClient } from '@/lib/ambient-client';
import { sessionKeys } from './v1/keys';
import type { AgenticSession } from '@/types/agentic-session';

export function useSessionDual(projectName: string, sessionId: string) {
  const { isApiServer } = useApiSource();

  const k8sResult = useSession(projectName, sessionId, { enabled: !isApiServer });
  const v1Result = useV1Session(projectName, sessionId, { enabled: isApiServer });

  const adaptedV1Data = useMemo<AgenticSession | undefined>(() => {
    if (!isApiServer || !v1Result.data) return undefined;
    return v1SessionToAgenticSession(v1Result.data);
  }, [isApiServer, v1Result.data]);

  if (isApiServer) {
    return {
      data: adaptedV1Data,
      isLoading: v1Result.isLoading,
      error: v1Result.error,
      refetch: v1Result.refetch,
    };
  }

  return {
    data: k8sResult.data,
    isLoading: k8sResult.isLoading,
    error: k8sResult.error,
    refetch: k8sResult.refetch,
  };
}

export function useStopSessionDual() {
  const { isApiServer } = useApiSource();
  const k8sMutation = useStopSession();
  const queryClient = useQueryClient();

  const v1Mutation = useMutation({
    mutationFn: async ({ projectName, sessionName }: { projectName: string; sessionName: string }) => {
      return createAmbientClient(projectName).sessions.stop(sessionName);
    },
    onSuccess: (_result, { sessionName }) => {
      queryClient.invalidateQueries({ queryKey: sessionKeys.detail(sessionName) });
      queryClient.invalidateQueries({ queryKey: sessionKeys.lists() });
    },
  });

  return isApiServer ? v1Mutation : k8sMutation;
}

export function useDeleteSessionDual() {
  const { isApiServer } = useApiSource();
  const k8sMutation = useDeleteSession();
  const queryClient = useQueryClient();

  const v1Mutation = useMutation({
    mutationFn: async ({ projectName, sessionName }: { projectName: string; sessionName: string }) => {
      const baseUrl = process.env.NEXT_PUBLIC_AMBIENT_API_URL || 'http://localhost:8000/api/ambient-api-server/v1';
      const resp = await fetch(`${baseUrl}/sessions/${sessionName}`, {
        method: 'DELETE',
        headers: {
          'Authorization': `Bearer no-auth`,
          'X-Ambient-Project': projectName || 'default',
        },
      });
      if (!resp.ok && resp.status !== 204) {
        throw new Error(`Delete failed: ${resp.status}`);
      }
    },
    onSuccess: (_result, { sessionName }) => {
      queryClient.removeQueries({ queryKey: sessionKeys.detail(sessionName) });
      queryClient.invalidateQueries({ queryKey: sessionKeys.lists() });
    },
  });

  return isApiServer ? v1Mutation : k8sMutation;
}

export function useContinueSessionDual() {
  const { isApiServer } = useApiSource();
  const k8sMutation = useContinueSession();
  const queryClient = useQueryClient();

  const v1Mutation = useMutation({
    mutationFn: async ({ projectName, parentSessionName }: { projectName: string; parentSessionName: string }) => {
      return createAmbientClient(projectName).sessions.start(parentSessionName);
    },
    onSuccess: (_result, { parentSessionName }) => {
      queryClient.invalidateQueries({ queryKey: sessionKeys.detail(parentSessionName) });
      queryClient.invalidateQueries({ queryKey: sessionKeys.lists() });
    },
  });

  return isApiServer ? v1Mutation : k8sMutation;
}
