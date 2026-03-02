import { useQuery } from '@tanstack/react-query';
import * as modelsApi from '@/services/api/models';

export const modelKeys = {
  forProject: (projectName: string) => ['models', projectName] as const,
};

export function useModels(projectName: string, enabled = true) {
  return useQuery({
    queryKey: modelKeys.forProject(projectName),
    queryFn: () => modelsApi.getModelsForProject(projectName),
    enabled: !!projectName && enabled,
    staleTime: 60_000,
  });
}
