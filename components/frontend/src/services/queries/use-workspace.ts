import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { sessionWorkspaceAdapter } from '../adapters/session-workspace';
import type { SessionWorkspacePort } from '../ports/session-workspace';
import { BACKEND_VERSION } from './query-keys';
import { sessionKeys } from './use-sessions';

export const workspaceKeys = {
  all: [BACKEND_VERSION, 'workspace'] as const,
  lists: () => [...workspaceKeys.all, 'list'] as const,
  list: (projectName: string, sessionName: string, path?: string) =>
    [...workspaceKeys.lists(), projectName, sessionName, path] as const,
  files: () => [...workspaceKeys.all, 'file'] as const,
  file: (projectName: string, sessionName: string, path: string) =>
    [...workspaceKeys.files(), projectName, sessionName, path] as const,
  diffs: () => [...workspaceKeys.all, 'diff'] as const,
  diff: (projectName: string, sessionName: string, repoIndex: number) =>
    [...workspaceKeys.diffs(), projectName, sessionName, repoIndex] as const,
};

export function useWorkspaceList(
  projectName: string,
  sessionName: string,
  path?: string,
  options?: { enabled?: boolean },
  port: SessionWorkspacePort = sessionWorkspaceAdapter,
) {
  return useQuery({
    queryKey: workspaceKeys.list(projectName, sessionName, path),
    queryFn: () => port.listWorkspace(projectName, sessionName, path),
    enabled: !!projectName && !!sessionName && (options?.enabled ?? true),
    staleTime: 5 * 1000,
  });
}

export function useWorkspaceFile(
  projectName: string,
  sessionName: string,
  path: string,
  options?: { enabled?: boolean; refetchInterval?: number | false; refetchOnMount?: boolean },
  port: SessionWorkspacePort = sessionWorkspaceAdapter,
) {
  return useQuery({
    queryKey: workspaceKeys.file(projectName, sessionName, path),
    queryFn: () => port.readFile(projectName, sessionName, path),
    enabled: !!projectName && !!sessionName && !!path && (options?.enabled ?? true),
    staleTime: 10 * 1000,
    refetchInterval: options?.refetchInterval,
    refetchOnMount: options?.refetchOnMount,
  });
}

export function useWriteWorkspaceFile(port: SessionWorkspacePort = sessionWorkspaceAdapter) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      projectName,
      sessionName,
      path,
      content,
    }: {
      projectName: string;
      sessionName: string;
      path: string;
      content: string;
    }) => port.writeFile(projectName, sessionName, path, content),
    onSuccess: (_data, { projectName, sessionName, path }) => {
      queryClient.invalidateQueries({
        queryKey: workspaceKeys.file(projectName, sessionName, path),
      });
      const parentPath = path.split('/').slice(0, -1).join('/');
      queryClient.invalidateQueries({
        queryKey: workspaceKeys.list(projectName, sessionName, parentPath || undefined),
      });
    },
  });
}

export function useSessionGitHubDiff(
  projectName: string,
  sessionName: string,
  repoIndex: number,
  repoPath: string,
  options?: { enabled?: boolean },
  port: SessionWorkspacePort = sessionWorkspaceAdapter,
) {
  return useQuery({
    queryKey: workspaceKeys.diff(projectName, sessionName, repoIndex),
    queryFn: () => port.getGitHubDiff(projectName, sessionName, repoIndex, repoPath),
    enabled: !!projectName && !!sessionName && (options?.enabled ?? true),
    staleTime: 10 * 1000,
  });
}

export function useAllSessionGitHubDiffs(
  projectName: string,
  sessionName: string,
  repos: Array<{ input: { url: string; branch: string }; output?: { url: string; branch: string } }> | undefined,
  deriveRepoFolder: (url: string) => string,
  options?: { enabled?: boolean; sessionPhase?: string },
  port: SessionWorkspacePort = sessionWorkspaceAdapter,
) {
  const queryClient = useQueryClient();

  return useQuery({
    queryKey: [...workspaceKeys.diffs(), projectName, sessionName, 'all'],
    queryFn: async () => {
      if (!repos || repos.length === 0) return {};

      const diffs = await Promise.all(
        repos.map(async (repo, idx) => {
          const url = repo?.input?.url || '';
          if (!url) return { idx, diff: { files: { added: 0, removed: 0 }, total_added: 0, total_removed: 0 } };

          const folder = deriveRepoFolder(url);
          const repoPath = `/sessions/${sessionName}/workspace/${folder}`;

          try {
            const diff = await queryClient.fetchQuery({
              queryKey: workspaceKeys.diff(projectName, sessionName, idx),
              queryFn: () => port.getGitHubDiff(projectName, sessionName, idx, repoPath),
            });
            return { idx, diff };
          } catch {
            return { idx, diff: { files: { added: 0, removed: 0 }, total_added: 0, total_removed: 0 } };
          }
        })
      );

      const totals: Record<number, { files: { added: number; removed: number }; total_added: number; total_removed: number }> = {};
      diffs.forEach(({ idx, diff }) => {
        totals[idx] = diff;
      });
      return totals;
    },
    enabled: !!projectName && !!sessionName && !!repos && (options?.enabled ?? true),
    staleTime: 10 * 1000,
    refetchInterval: () => {
      const phase = options?.sessionPhase;
      const isTransitioning =
        phase === 'Stopping' ||
        phase === 'Pending' ||
        phase === 'Creating';
      if (isTransitioning) return 5000;

      if (phase === 'Running') return 10000;

      return false;
    },
  });
}

export function usePushSessionToGitHub(port: SessionWorkspacePort = sessionWorkspaceAdapter) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      projectName,
      sessionName,
      repoIndex,
      repoPath,
    }: {
      projectName: string;
      sessionName: string;
      repoIndex: number;
      repoPath: string;
    }) => port.pushToGitHub(projectName, sessionName, repoIndex, repoPath),
    onSuccess: (_data, { projectName, sessionName, repoIndex }) => {
      queryClient.invalidateQueries({
        queryKey: workspaceKeys.diff(projectName, sessionName, repoIndex),
      });
      queryClient.invalidateQueries({
        queryKey: sessionKeys.detail(projectName, sessionName),
      });
    },
  });
}

export function useAbandonSessionChanges(port: SessionWorkspacePort = sessionWorkspaceAdapter) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      projectName,
      sessionName,
      repoIndex,
      repoPath,
    }: {
      projectName: string;
      sessionName: string;
      repoIndex: number;
      repoPath: string;
    }) => port.abandonChanges(projectName, sessionName, repoIndex, repoPath),
    onSuccess: (_data, { projectName, sessionName, repoIndex }) => {
      queryClient.invalidateQueries({
        queryKey: workspaceKeys.diff(projectName, sessionName, repoIndex),
      });
      queryClient.invalidateQueries({
        queryKey: workspaceKeys.lists(),
      });
    },
  });
}

export function useGitMergeStatus(
  projectName: string,
  sessionName: string,
  path: string = 'artifacts',
  branch: string = 'main',
  enabled: boolean = true,
  port: SessionWorkspacePort = sessionWorkspaceAdapter,
) {
  return useQuery({
    queryKey: [...workspaceKeys.all, 'git-merge-status', projectName, sessionName, path, branch],
    queryFn: () => port.getGitMergeStatus(projectName, sessionName, path, branch),
    enabled: enabled && !!projectName && !!sessionName,
    staleTime: 5000,
  });
}

export function useGitCreateBranch(port: SessionWorkspacePort = sessionWorkspaceAdapter) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      projectName,
      sessionName,
      branchName,
      path = 'artifacts',
    }: {
      projectName: string;
      sessionName: string;
      branchName: string;
      path?: string;
    }) => port.gitCreateBranch(projectName, sessionName, branchName, path),
    onSuccess: (_data, { projectName, sessionName }) => {
      queryClient.invalidateQueries({
        queryKey: [...workspaceKeys.all, 'git-branches', projectName, sessionName],
      });
      queryClient.invalidateQueries({
        queryKey: [...workspaceKeys.all, 'git-merge-status', projectName, sessionName],
      });
    },
  });
}

export function useGitListBranches(
  projectName: string,
  sessionName: string,
  path: string = 'artifacts',
  enabled: boolean = true,
  port: SessionWorkspacePort = sessionWorkspaceAdapter,
) {
  return useQuery({
    queryKey: [...workspaceKeys.all, 'git-branches', projectName, sessionName, path],
    queryFn: () => port.gitListBranches(projectName, sessionName, path),
    enabled: enabled && !!projectName && !!sessionName,
    staleTime: 30000,
  });
}

export function useGitStatus(
  projectName: string,
  sessionName: string,
  path: string,
  options?: { enabled?: boolean },
  port: SessionWorkspacePort = sessionWorkspaceAdapter,
) {
  return useQuery({
    queryKey: [...workspaceKeys.all, 'git-status', projectName, sessionName, path],
    queryFn: () => port.gitStatus(projectName, sessionName, path),
    enabled: !!projectName && !!sessionName && !!path && (options?.enabled ?? true),
    staleTime: 5000,
  });
}

export function useConfigureGitRemote(port: SessionWorkspacePort = sessionWorkspaceAdapter) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      projectName,
      sessionName,
      path,
      remoteUrl,
      branch = 'main',
    }: {
      projectName: string;
      sessionName: string;
      path: string;
      remoteUrl: string;
      branch?: string;
    }) => port.configureGitRemote(projectName, sessionName, path, remoteUrl, branch),
    onSuccess: (_data, { projectName, sessionName, path }) => {
      queryClient.invalidateQueries({
        queryKey: [...workspaceKeys.all, 'git-status', projectName, sessionName, path],
      });
      queryClient.invalidateQueries({
        queryKey: [...workspaceKeys.all, 'git-branches', projectName, sessionName, path],
      });
    },
  });
}
