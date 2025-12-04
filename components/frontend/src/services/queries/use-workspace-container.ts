import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import * as workspaceContainerApi from '../api/workspace-container';
import type { WorkspaceContainerSettings } from '@/types/project-settings';

export function useWorkspaceContainerSettings(projectName: string) {
  return useQuery({
    queryKey: ['workspace-container', projectName],
    queryFn: () => workspaceContainerApi.getWorkspaceContainerSettings(projectName),
    enabled: !!projectName,
  });
}

export function useUpdateWorkspaceContainerSettings() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      projectName,
      settings,
    }: {
      projectName: string;
      settings: WorkspaceContainerSettings;
    }) => workspaceContainerApi.updateWorkspaceContainerSettings(projectName, settings),
    onSuccess: (_, { projectName }) => {
      queryClient.invalidateQueries({ queryKey: ['workspace-container', projectName] });
    },
    onError: (error) => {
      console.error('Failed to update workspace container settings:', error);
    },
  });
}
