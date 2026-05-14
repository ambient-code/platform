import { useQuery } from '@tanstack/react-query';
import { modelsAdapter } from '../adapters/models';
import type { ModelsPort } from '../ports/models';
import { BACKEND_VERSION } from './query-keys';

export const modelKeys = {
  forProject: (projectName: string, provider?: string) =>
    [BACKEND_VERSION, 'models', projectName, ...(provider ? [provider] : [])] as const,
};

export function useModels(projectName: string, enabled = true, provider?: string, port: ModelsPort = modelsAdapter) {
  return useQuery({
    queryKey: modelKeys.forProject(projectName, provider),
    queryFn: () => port.getModelsForProject(projectName, provider),
    enabled: !!projectName && enabled,
    staleTime: 60_000,
  });
}
