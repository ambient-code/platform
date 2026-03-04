import { useQuery } from '@tanstack/react-query';
import * as modelsApi from '@/services/api/models';

export const modelKeys = {
  forProject: (projectName: string, provider?: string) =>
    ['models', projectName, ...(provider ? [provider] : [])] as const,
};

export function useModels(projectName: string, enabled = true, provider?: string) {
  return useQuery({
    queryKey: modelKeys.forProject(projectName, provider),
    queryFn: () => modelsApi.getModelsForProject(projectName, provider),
    enabled: !!projectName && enabled,
    staleTime: 60_000,
  });
}
