import { useMutation, useQuery, useQueryClient, keepPreviousData } from '@tanstack/react-query';
import type { TaskCreateRequest, TaskPatchRequest, ListOptions } from '@ambient-platform/sdk';
import { createAmbientClient } from '@/lib/ambient-client';
import { taskKeys } from './keys';

export function useV1Tasks(project: string, opts?: ListOptions) {
  return useQuery({
    queryKey: taskKeys.list(opts),
    queryFn: () => createAmbientClient(project).tasks.list(opts),
    enabled: !!project,
    placeholderData: keepPreviousData,
  });
}

export function useV1Task(project: string, id: string) {
  return useQuery({
    queryKey: taskKeys.detail(id),
    queryFn: () => createAmbientClient(project).tasks.get(id),
    enabled: !!project && !!id,
  });
}

export function useV1CreateTask(project: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: TaskCreateRequest) =>
      createAmbientClient(project).tasks.create(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: taskKeys.lists() });
    },
  });
}

export function useV1UpdateTask(project: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: TaskPatchRequest }) =>
      createAmbientClient(project).tasks.update(id, data),
    onSuccess: (_result, { id }) => {
      queryClient.invalidateQueries({ queryKey: taskKeys.detail(id) });
      queryClient.invalidateQueries({ queryKey: taskKeys.lists() });
    },
  });
}
