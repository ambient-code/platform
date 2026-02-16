import { useMutation, useQuery, useQueryClient, keepPreviousData } from '@tanstack/react-query';
import type { WorkflowTaskCreateRequest, WorkflowTaskPatchRequest, ListOptions } from '@ambient-platform/sdk';
import { createAmbientClient } from '@/lib/ambient-client';
import { workflowTaskKeys } from './keys';

export function useV1WorkflowTasks(project: string, opts?: ListOptions) {
  return useQuery({
    queryKey: workflowTaskKeys.list(opts),
    queryFn: () => createAmbientClient(project).workflowTasks.list(opts),
    enabled: !!project,
    placeholderData: keepPreviousData,
  });
}

export function useV1WorkflowTask(project: string, id: string) {
  return useQuery({
    queryKey: workflowTaskKeys.detail(id),
    queryFn: () => createAmbientClient(project).workflowTasks.get(id),
    enabled: !!project && !!id,
  });
}

export function useV1CreateWorkflowTask(project: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: WorkflowTaskCreateRequest) =>
      createAmbientClient(project).workflowTasks.create(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: workflowTaskKeys.lists() });
    },
  });
}

export function useV1UpdateWorkflowTask(project: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: WorkflowTaskPatchRequest }) =>
      createAmbientClient(project).workflowTasks.update(id, data),
    onSuccess: (_result, { id }) => {
      queryClient.invalidateQueries({ queryKey: workflowTaskKeys.detail(id) });
      queryClient.invalidateQueries({ queryKey: workflowTaskKeys.lists() });
    },
  });
}
