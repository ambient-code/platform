import { useMutation, useQuery, useQueryClient, keepPreviousData } from '@tanstack/react-query';
import type { SkillCreateRequest, SkillPatchRequest, ListOptions } from '@ambient-platform/sdk';
import { createAmbientClient } from '@/lib/ambient-client';
import { skillKeys } from './keys';

export function useV1Skills(project: string, opts?: ListOptions) {
  return useQuery({
    queryKey: skillKeys.list(opts),
    queryFn: () => createAmbientClient(project).skills.list(opts),
    enabled: !!project,
    placeholderData: keepPreviousData,
  });
}

export function useV1Skill(project: string, id: string) {
  return useQuery({
    queryKey: skillKeys.detail(id),
    queryFn: () => createAmbientClient(project).skills.get(id),
    enabled: !!project && !!id,
  });
}

export function useV1CreateSkill(project: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: SkillCreateRequest) =>
      createAmbientClient(project).skills.create(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: skillKeys.lists() });
    },
  });
}

export function useV1UpdateSkill(project: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: SkillPatchRequest }) =>
      createAmbientClient(project).skills.update(id, data),
    onSuccess: (_result, { id }) => {
      queryClient.invalidateQueries({ queryKey: skillKeys.detail(id) });
      queryClient.invalidateQueries({ queryKey: skillKeys.lists() });
    },
  });
}
