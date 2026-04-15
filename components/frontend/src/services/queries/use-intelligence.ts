/**
 * React Query hooks for repo intelligence
 */

import { useQuery } from '@tanstack/react-query';
import { getRepoIntelligence } from '../api/intelligence';

export const intelligenceKeys = {
  all: ['intelligence'] as const,
  repo: (projectName: string, repoUrl: string) =>
    [...intelligenceKeys.all, 'repo', projectName, repoUrl] as const,
};

/**
 * Fetch intelligence for a specific repo in a project.
 * Returns null data when no intelligence exists (not an error).
 */
export function useRepoIntelligence(
  projectName: string,
  repoUrl: string,
  enabled = true
) {
  return useQuery({
    queryKey: intelligenceKeys.repo(projectName, repoUrl),
    queryFn: () => getRepoIntelligence(projectName, repoUrl),
    enabled: enabled && !!projectName && !!repoUrl,
    staleTime: 60_000,
    retry: false,
  });
}
