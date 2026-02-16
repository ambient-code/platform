import { useMutation, useQuery, useQueryClient, keepPreviousData } from '@tanstack/react-query';
import type { WorkflowSkillCreateRequest, WorkflowSkillPatchRequest, ListOptions } from '@ambient-platform/sdk';
import { createAmbientClient } from '@/lib/ambient-client';
import { workflowSkillKeys } from './keys';

export function useV1WorkflowSkills(project: string, opts?: ListOptions) {
  return useQuery({
    queryKey: workflowSkillKeys.list(opts),
    queryFn: () => createAmbientClient(project).workflowSkills.list(opts),
    enabled: !!project,
    placeholderData: keepPreviousData,
  });
}

export function useV1WorkflowSkill(project: string, id: string) {
  return useQuery({
    queryKey: workflowSkillKeys.detail(id),
    queryFn: () => createAmbientClient(project).workflowSkills.get(id),
    enabled: !!project && !!id,
  });
}

export function useV1CreateWorkflowSkill(project: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: WorkflowSkillCreateRequest) =>
      createAmbientClient(project).workflowSkills.create(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: workflowSkillKeys.lists() });
    },
  });
}

export function useV1UpdateWorkflowSkill(project: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: WorkflowSkillPatchRequest }) =>
      createAmbientClient(project).workflowSkills.update(id, data),
    onSuccess: (_result, { id }) => {
      queryClient.invalidateQueries({ queryKey: workflowSkillKeys.detail(id) });
      queryClient.invalidateQueries({ queryKey: workflowSkillKeys.lists() });
    },
  });
}
