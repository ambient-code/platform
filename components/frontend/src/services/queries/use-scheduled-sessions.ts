import { useMutation, useQuery, useQueryClient, keepPreviousData } from '@tanstack/react-query';
import { scheduledSessionsAdapter } from '../adapters/scheduled-sessions';
import type { ScheduledSessionsPort } from '../ports/scheduled-sessions';
import type {
  CreateScheduledSessionRequest,
  UpdateScheduledSessionRequest,
} from '@/types/api';
import { BACKEND_VERSION } from './query-keys';

export const scheduledSessionKeys = {
  all: [BACKEND_VERSION, 'scheduled-sessions'] as const,
  lists: () => [...scheduledSessionKeys.all, 'list'] as const,
  list: (projectName: string) =>
    [...scheduledSessionKeys.lists(), projectName] as const,
  details: () => [...scheduledSessionKeys.all, 'detail'] as const,
  detail: (projectName: string, name: string) =>
    [...scheduledSessionKeys.details(), projectName, name] as const,
  runs: (projectName: string, name: string) =>
    [...scheduledSessionKeys.detail(projectName, name), 'runs'] as const,
};

export function useScheduledSessions(projectName: string, port: ScheduledSessionsPort = scheduledSessionsAdapter) {
  return useQuery({
    queryKey: scheduledSessionKeys.list(projectName),
    queryFn: () => port.listScheduledSessions(projectName),
    enabled: !!projectName,
    placeholderData: keepPreviousData,
  });
}

export function useScheduledSession(projectName: string, name: string, port: ScheduledSessionsPort = scheduledSessionsAdapter) {
  return useQuery({
    queryKey: scheduledSessionKeys.detail(projectName, name),
    queryFn: () => port.getScheduledSession(projectName, name),
    enabled: !!projectName && !!name,
  });
}

export function useScheduledSessionRuns(projectName: string, name: string, port: ScheduledSessionsPort = scheduledSessionsAdapter) {
  return useQuery({
    queryKey: scheduledSessionKeys.runs(projectName, name),
    queryFn: () => port.listScheduledSessionRuns(projectName, name),
    enabled: !!projectName && !!name,
  });
}

export function useCreateScheduledSession(port: ScheduledSessionsPort = scheduledSessionsAdapter) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      projectName,
      data,
    }: {
      projectName: string;
      data: CreateScheduledSessionRequest;
    }) => port.createScheduledSession(projectName, data),
    onSuccess: (_result, { projectName }) => {
      queryClient.invalidateQueries({
        queryKey: scheduledSessionKeys.list(projectName),
        refetchType: 'all',
      });
    },
  });
}

export function useUpdateScheduledSession(port: ScheduledSessionsPort = scheduledSessionsAdapter) {
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
    }) => port.updateScheduledSession(projectName, name, data),
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

export function useDeleteScheduledSession(port: ScheduledSessionsPort = scheduledSessionsAdapter) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      projectName,
      name,
    }: {
      projectName: string;
      name: string;
    }) => port.deleteScheduledSession(projectName, name),
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

export function useSuspendScheduledSession(port: ScheduledSessionsPort = scheduledSessionsAdapter) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      projectName,
      name,
    }: {
      projectName: string;
      name: string;
    }) => port.suspendScheduledSession(projectName, name),
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

export function useResumeScheduledSession(port: ScheduledSessionsPort = scheduledSessionsAdapter) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      projectName,
      name,
    }: {
      projectName: string;
      name: string;
    }) => port.resumeScheduledSession(projectName, name),
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

export function useTriggerScheduledSession(port: ScheduledSessionsPort = scheduledSessionsAdapter) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      projectName,
      name,
    }: {
      projectName: string;
      name: string;
    }) => port.triggerScheduledSession(projectName, name),
    onSuccess: (_result, { projectName, name }) => {
      queryClient.invalidateQueries({
        queryKey: scheduledSessionKeys.runs(projectName, name),
        refetchType: 'all',
      });
    },
  });
}
