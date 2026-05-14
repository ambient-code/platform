import { useQuery } from '@tanstack/react-query';
import { projectAccessAdapter } from '../adapters/project-access';
import type { ProjectAccessPort } from '../ports/project-access';
import { BACKEND_VERSION } from './query-keys';

export const projectAccessKeys = {
  all: [BACKEND_VERSION, 'project-access'] as const,
  forProject: (projectName: string) => [...projectAccessKeys.all, projectName] as const,
};

export function useProjectAccess(projectName: string, port: ProjectAccessPort = projectAccessAdapter) {
  return useQuery({
    queryKey: projectAccessKeys.forProject(projectName),
    queryFn: () => port.getAccess(projectName),
    enabled: !!projectName,
    staleTime: 60000,
    retry: 1,
  });
}
