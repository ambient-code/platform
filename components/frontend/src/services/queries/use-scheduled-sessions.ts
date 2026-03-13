/**
 * React Query hooks for scheduled sessions
 */

import { useMutation, useQuery, useQueryClient, keepPreviousData } from '@tanstack/react-query';
import * as scheduledSessionsApi from '../api/scheduled-sessions';
import type {
  CreateScheduledSessionRequest,
  UpdateScheduledSessionRequest,
} from '@/types/api';

export const scheduledSessionKeys = {
  all: ['scheduled-sessions'] as const,
  lists: () => [...scheduledSessionKeys.all, 'list'] as const,
  list: (projectName: string) =>
    [...scheduledSessionKeys.lists(), projectName] as const,
  details: () => [...scheduledSessionKeys.all, 'detail'] as const,
  detail: (projectName: string, name: string) =>
    [...scheduledSessionKeys.details(), projectName, name] as const,
  runs: (projectName: string, name: string) =>
    [...scheduledSessionKeys.detail(projectName, name), 'runs'] as const,
};

export function useScheduledSessions(projectName: string) {
  return useQuery({
    queryKey: scheduledSessionKeys.list(projectName),
    queryFn: () => scheduledSessionsApi.listScheduledSessions(projectName),
    enabled: !!projectName,
    placeholderData: keepPreviousData,
  });
}

export function useScheduledSession(projectName: string, name: string) {
  return useQuery({
    queryKey: scheduledSessionKeys.detail(projectName, name),
    queryFn: () => scheduledSessionsApi.getScheduledSession(projectName, name),
    enabled: !!projectName && !!name,
  });
}

export function useScheduledSessionRuns(projectName: string, name: string) {
  return useQuery({
    queryKey: scheduledSessionKeys.runs(projectName, name),
    queryFn: () => scheduledSessionsApi.listScheduledSessionRuns(projectName, name),
    enabled: !!projectName && !!name,
  });
}

export function useCreateScheduledSession() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      projectName,
      data,
    }: {
      projectName: string;
      data: CreateScheduledSessionRequest;
    }) => scheduledSessionsApi.createScheduledSession(projectName, data),
    onSuccess: (_result, { projectName }) => {
      queryClient.invalidateQueries({
        queryKey: scheduledSessionKeys.list(projectName),
        refetchType: 'all',
      });
    },
  });
}

export function useUpdateScheduledSession() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      projectName,
      name,
      data,
    }: {
      projectName: string;
      name: string;
      data: UpdateScheduledSessionRequest;
    }) => scheduledSessionsApi.updateScheduledSession(projectName, name, data),
    onSuccess: (_result, { projectName, name }) => {
      queryClient.invalidateQueries({
        queryKey: scheduledSessionKeys.detail(projectName, name),
        refetchType: 'all',
      });
      queryClient.invalidateQueries({
        queryKey: scheduledSessionKeys.list(projectName),
        refetchType: 'all',
      });
    },
  });
}

export function useDeleteScheduledSession() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      projectName,
      name,
    }: {
      projectName: string;
      name: string;
    }) => scheduledSessionsApi.deleteScheduledSession(projectName, name),
    onSuccess: (_data, { projectName, name }) => {
      queryClient.removeQueries({
        queryKey: scheduledSessionKeys.detail(projectName, name),
      });
      queryClient.invalidateQueries({
        queryKey: scheduledSessionKeys.list(projectName),
        refetchType: 'all',
      });
    },
  });
}

export function useSuspendScheduledSession() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      projectName,
      name,
    }: {
      projectName: string;
      name: string;
    }) => scheduledSessionsApi.suspendScheduledSession(projectName, name),
    onSuccess: (_result, { projectName, name }) => {
      queryClient.invalidateQueries({
        queryKey: scheduledSessionKeys.detail(projectName, name),
        refetchType: 'all',
      });
      queryClient.invalidateQueries({
        queryKey: scheduledSessionKeys.list(projectName),
        refetchType: 'all',
      });
    },
  });
}

export function useResumeScheduledSession() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      projectName,
      name,
    }: {
      projectName: string;
      name: string;
    }) => scheduledSessionsApi.resumeScheduledSession(projectName, name),
    onSuccess: (_result, { projectName, name }) => {
      queryClient.invalidateQueries({
        queryKey: scheduledSessionKeys.detail(projectName, name),
        refetchType: 'all',
      });
      queryClient.invalidateQueries({
        queryKey: scheduledSessionKeys.list(projectName),
        refetchType: 'all',
      });
    },
  });
}

export function useTriggerScheduledSession() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      projectName,
      name,
    }: {
      projectName: string;
      name: string;
    }) => scheduledSessionsApi.triggerScheduledSession(projectName, name),
    onSuccess: (_result, { projectName, name }) => {
      queryClient.invalidateQueries({
        queryKey: scheduledSessionKeys.runs(projectName, name),
        refetchType: 'all',
      });
    },
  });
}
