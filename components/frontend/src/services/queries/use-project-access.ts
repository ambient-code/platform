import { useQuery } from '@tanstack/react-query';
import { projectAccessAdapter } from '../adapters/project-access';
import type { ProjectAccessPort } from '../ports/project-access';

export function useProjectAccess(projectName: string, port: ProjectAccessPort = projectAccessAdapter) {
  return useQuery({
    queryKey: ['project-access', projectName],
    queryFn: () => port.getAccess(projectName),
    enabled: !!projectName,
    staleTime: 60000,
    retry: 1,
  });
}
