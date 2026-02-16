import { useMutation, useQuery, useQueryClient, keepPreviousData } from '@tanstack/react-query';
import type {
  ProjectSettingsCreateRequest,
  ProjectSettingsPatchRequest,
  ListOptions,
} from '@ambient-platform/sdk';
import { createAmbientClient } from '@/lib/ambient-client';
import { projectSettingsKeys } from './keys';

export function useV1ProjectSettingsList(project: string, opts?: ListOptions) {
  return useQuery({
    queryKey: projectSettingsKeys.list(opts),
    queryFn: () => createAmbientClient(project).projectSettings.list(opts),
    enabled: !!project,
    placeholderData: keepPreviousData,
  });
}

export function useV1ProjectSettings(project: string, id: string) {
  return useQuery({
    queryKey: projectSettingsKeys.detail(id),
    queryFn: () => createAmbientClient(project).projectSettings.get(id),
    enabled: !!project && !!id,
  });
}

export function useV1CreateProjectSettings(project: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: ProjectSettingsCreateRequest) =>
      createAmbientClient(project).projectSettings.create(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: projectSettingsKeys.lists() });
    },
  });
}

export function useV1UpdateProjectSettings(project: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: ProjectSettingsPatchRequest }) =>
      createAmbientClient(project).projectSettings.update(id, data),
    onSuccess: (_result, { id }) => {
      queryClient.invalidateQueries({ queryKey: projectSettingsKeys.detail(id) });
      queryClient.invalidateQueries({ queryKey: projectSettingsKeys.lists() });
    },
  });
}

export function useV1DeleteProjectSettings(project: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) =>
      createAmbientClient(project).projectSettings.delete(id),
    onSuccess: (_result, id) => {
      queryClient.removeQueries({ queryKey: projectSettingsKeys.detail(id) });
      queryClient.invalidateQueries({ queryKey: projectSettingsKeys.lists() });
    },
  });
}
