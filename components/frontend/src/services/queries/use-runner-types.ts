import { useQuery } from '@tanstack/react-query';
import { runnerTypesAdapter } from '../adapters/runner-types';
import type { RunnerTypesPort } from '../ports/runner-types';

export const runnerTypeKeys = {
  all: ['runner-types'] as const,
  global: () => [...runnerTypeKeys.all, 'global'] as const,
  forProject: (projectName: string) => [...runnerTypeKeys.all, projectName] as const,
};

export function useRunnerTypes(projectName: string, port: RunnerTypesPort = runnerTypesAdapter) {
  return useQuery({
    queryKey: runnerTypeKeys.forProject(projectName),
    queryFn: () => port.getRunnerTypes(projectName),
    enabled: !!projectName,
    staleTime: 5 * 60 * 1000,
    gcTime: 30 * 60 * 1000,
  });
}
