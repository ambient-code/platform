import { useMutation, useQuery, useQueryClient, keepPreviousData } from '@tanstack/react-query';
import type { WorkflowCreateRequest, WorkflowPatchRequest, ListOptions } from '@ambient-platform/sdk';
import { createAmbientClient } from '@/lib/ambient-client';
import { workflowKeys } from './keys';

export function useV1Workflows(project: string, opts?: ListOptions) {
  return useQuery({
    queryKey: workflowKeys.list(opts),
    queryFn: () => createAmbientClient(project).workflows.list(opts),
    enabled: !!project,
    placeholderData: keepPreviousData,
  });
}

export function useV1Workflow(project: string, id: string) {
  return useQuery({
    queryKey: workflowKeys.detail(id),
    queryFn: () => createAmbientClient(project).workflows.get(id),
    enabled: !!project && !!id,
  });
}

export function useV1CreateWorkflow(project: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: WorkflowCreateRequest) =>
      createAmbientClient(project).workflows.create(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: workflowKeys.lists() });
    },
  });
}

export function useV1UpdateWorkflow(project: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: WorkflowPatchRequest }) =>
      createAmbientClient(project).workflows.update(id, data),
    onSuccess: (_result, { id }) => {
      queryClient.invalidateQueries({ queryKey: workflowKeys.detail(id) });
      queryClient.invalidateQueries({ queryKey: workflowKeys.lists() });
    },
  });
}
