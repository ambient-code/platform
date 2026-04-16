/**
 * React Query hooks for learned files and draft PRs
 */

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import * as learnedApi from '../api/learned';

/**
 * Query keys for learned files
 */
export const learnedKeys = {
  all: ['learned'] as const,
  lists: () => [...learnedKeys.all, 'list'] as const,
  list: (projectName: string, type?: string, page?: number) =>
    [...learnedKeys.lists(), projectName, type, page] as const,
  prs: () => [...learnedKeys.all, 'prs'] as const,
  prList: (projectName: string) =>
    [...learnedKeys.prs(), projectName] as const,
};

/**
 * Hook to fetch learned files for a project
 */
export function useLearnedFiles(
  projectName: string,
  params?: { type?: string; page?: number; pageSize?: number },
  options?: { enabled?: boolean }
) {
  return useQuery({
    queryKey: learnedKeys.list(projectName, params?.type, params?.page),
    queryFn: () => learnedApi.getLearnedFiles(projectName, params),
    enabled: (options?.enabled ?? true) && !!projectName,
    staleTime: 60 * 1000, // 1 minute
  });
}

/**
 * Hook to fetch draft PRs with continuous-learning label
 */
export function useLearnedDraftPRs(
  projectName: string,
  options?: { enabled?: boolean }
) {
  return useQuery({
    queryKey: learnedKeys.prList(projectName),
    queryFn: () => learnedApi.getLearnedDraftPRs(projectName),
    enabled: (options?.enabled ?? true) && !!projectName,
    staleTime: 60 * 1000, // 1 minute
  });
}

/**
 * Hook to create a new memory via PR
 */
export function useCreateMemory() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      projectName,
      data,
    }: {
      projectName: string;
      data: learnedApi.CreateMemoryRequest;
    }) => learnedApi.createMemory(projectName, data),
    onSuccess: (_, { projectName }) => {
      queryClient.invalidateQueries({
        queryKey: learnedKeys.lists(),
      });
      queryClient.invalidateQueries({
        queryKey: learnedKeys.prList(projectName),
      });
    },
  });
}
