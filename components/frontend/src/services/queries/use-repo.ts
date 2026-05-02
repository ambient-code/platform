import { useQuery } from '@tanstack/react-query';
import { repoAdapter } from '../adapters/repo';
import type { RepoPort } from '../ports/repo';
import { BACKEND_VERSION } from './query-keys';

type RepoParams = {
  repo: string;
  ref: string;
  path: string;
};

export const repoKeys = {
  all: [BACKEND_VERSION, 'repo'] as const,
  blobs: () => [...repoKeys.all, 'blob'] as const,
  blob: (projectName: string, params: RepoParams) =>
    [...repoKeys.blobs(), projectName, params.repo, params.ref, params.path] as const,
  trees: () => [...repoKeys.all, 'tree'] as const,
  tree: (projectName: string, params: RepoParams) =>
    [...repoKeys.trees(), projectName, params.repo, params.ref, params.path] as const,
  branches: () => [...repoKeys.all, 'branches'] as const,
  repoBranches: (projectName: string, repo: string) =>
    [...repoKeys.branches(), projectName, repo] as const,
};

export function useRepoBlob(
  projectName: string,
  params: RepoParams,
  options?: { enabled?: boolean },
  port: RepoPort = repoAdapter,
) {
  return useQuery({
    queryKey: repoKeys.blob(projectName, params),
    queryFn: () => port.getRepoBlob(projectName, params),
    enabled: (options?.enabled ?? true) && !!projectName && !!params.repo && !!params.ref && !!params.path,
    staleTime: 5 * 60 * 1000,
  });
}

export function useRepoTree(
  projectName: string,
  params: RepoParams,
  options?: { enabled?: boolean },
  port: RepoPort = repoAdapter,
) {
  return useQuery({
    queryKey: repoKeys.tree(projectName, params),
    queryFn: () => port.getRepoTree(projectName, params),
    enabled: (options?.enabled ?? true) && !!projectName && !!params.repo && !!params.ref && !!params.path,
    staleTime: 5 * 60 * 1000,
  });
}

export function useRepoFileExists(
  projectName: string,
  params: RepoParams,
  options?: { enabled?: boolean },
  port: RepoPort = repoAdapter,
) {
  return useQuery({
    queryKey: [...repoKeys.blob(projectName, params), 'exists'] as const,
    queryFn: () => port.checkFileExists(projectName, params),
    enabled: (options?.enabled ?? true) && !!projectName && !!params.repo && !!params.ref && !!params.path,
    staleTime: 5 * 60 * 1000,
  });
}

export function useRepoBranches(
  projectName: string,
  repo: string,
  options?: { enabled?: boolean },
  port: RepoPort = repoAdapter,
) {
  return useQuery({
    queryKey: repoKeys.repoBranches(projectName, repo),
    queryFn: () => port.listRepoBranches(projectName, repo),
    enabled: (options?.enabled ?? true) && !!projectName && !!repo,
    staleTime: 2 * 60 * 1000,
  });
}
