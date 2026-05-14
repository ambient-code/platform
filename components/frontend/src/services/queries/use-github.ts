import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { githubAdapter } from '../adapters/github';
import type { GitHubPort } from '../ports/github';
import type {
  CreateForkRequest,
  CreatePRRequest,
  GitHubConnectRequest,
} from '@/types/api';
import { BACKEND_VERSION } from './query-keys';
import { integrationsKeys } from './use-integrations';

export const githubKeys = {
  all: [BACKEND_VERSION, 'github'] as const,
  status: () => [...githubKeys.all, 'status'] as const,
  forks: () => [...githubKeys.all, 'forks'] as const,
  forksForProject: (projectName: string, upstreamRepo?: string) =>
    [...githubKeys.forks(), projectName, upstreamRepo] as const,
  diff: (owner: string, repo: string, prNumber: number) =>
    [...githubKeys.all, 'diff', owner, repo, prNumber] as const,
};

export function useGitHubStatus(port: GitHubPort = githubAdapter) {
  return useQuery({
    queryKey: githubKeys.status(),
    queryFn: port.getGitHubStatus,
    staleTime: 60 * 1000,
  });
}

export function useGitHubForks(projectName?: string, upstreamRepo?: string, port: GitHubPort = githubAdapter) {
  return useQuery({
    queryKey: githubKeys.forksForProject(projectName || '', upstreamRepo),
    queryFn: () => port.listGitHubForks(projectName, upstreamRepo),
    enabled: !!projectName && !!upstreamRepo,
    staleTime: 5 * 60 * 1000,
  });
}

export function usePRDiff(
  owner: string,
  repo: string,
  prNumber: number,
  projectName?: string,
  port: GitHubPort = githubAdapter,
) {
  return useQuery({
    queryKey: githubKeys.diff(owner, repo, prNumber),
    queryFn: () => port.getPRDiff(owner, repo, prNumber, projectName),
    enabled: !!owner && !!repo && !!prNumber,
    staleTime: 60 * 1000,
  });
}

export function useConnectGitHub(port: GitHubPort = githubAdapter) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: GitHubConnectRequest) => port.connectGitHub(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: githubKeys.status() });
      queryClient.invalidateQueries({ queryKey: integrationsKeys.status() });
    },
  });
}

export function useDisconnectGitHub(port: GitHubPort = githubAdapter) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: port.disconnectGitHub,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: githubKeys.status() });
      queryClient.invalidateQueries({ queryKey: integrationsKeys.status() });
      queryClient.invalidateQueries({ queryKey: githubKeys.forks() });
    },
  });
}

export function useCreateGitHubFork(port: GitHubPort = githubAdapter) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      data,
      projectName,
    }: {
      data: CreateForkRequest;
      projectName?: string;
    }) => port.createGitHubFork(data, projectName),
    onSuccess: (_fork, { projectName }) => {
      if (projectName) {
        queryClient.invalidateQueries({
          queryKey: githubKeys.forksForProject(projectName),
        });
      } else {
        queryClient.invalidateQueries({ queryKey: githubKeys.forks() });
      }
    },
  });
}

export function useCreatePullRequest(port: GitHubPort = githubAdapter) {
  return useMutation({
    mutationFn: ({
      data,
      projectName,
    }: {
      data: CreatePRRequest;
      projectName?: string;
    }) => port.createPullRequest(data, projectName),
  });
}

export function useGitHubPATStatus(port: GitHubPort = githubAdapter) {
  return useQuery({
    queryKey: [...githubKeys.all, 'pat', 'status'],
    queryFn: port.getGitHubPATStatus,
    staleTime: 60 * 1000,
  });
}

export function useSaveGitHubPAT(port: GitHubPort = githubAdapter) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (token: string) => port.saveGitHubPAT(token),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: [...githubKeys.all, 'pat', 'status'] });
      queryClient.invalidateQueries({ queryKey: githubKeys.status() });
      queryClient.invalidateQueries({ queryKey: integrationsKeys.status() });
    },
  });
}

export function useDeleteGitHubPAT(port: GitHubPort = githubAdapter) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: port.deleteGitHubPAT,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: [...githubKeys.all, 'pat', 'status'] });
      queryClient.invalidateQueries({ queryKey: githubKeys.status() });
      queryClient.invalidateQueries({ queryKey: integrationsKeys.status() });
    },
  });
}
