/* eslint-disable react/display-name */
import { renderHook, waitFor, act } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { describe, it, expect, vi } from 'vitest';
import React from 'react';
import {
  useWorkspaceList,
  useWorkspaceFile,
  useWriteWorkspaceFile,
  useSessionGitHubDiff,
  useAllSessionGitHubDiffs,
  usePushSessionToGitHub,
  useAbandonSessionChanges,
  useGitMergeStatus,
  useGitCreateBranch,
  useGitListBranches,
  useGitStatus,
  useConfigureGitRemote,
  workspaceKeys,
} from '../use-workspace';

vi.mock('@/services/api/workspace', () => ({
  listWorkspace: vi.fn().mockResolvedValue([{ name: 'file.txt', type: 'file' }]),
  readWorkspaceFile: vi.fn().mockResolvedValue('file content'),
  writeWorkspaceFile: vi.fn().mockResolvedValue(undefined),
  getSessionGitHubDiff: vi.fn().mockResolvedValue({ files: { added: 1, removed: 0 }, total_added: 5, total_removed: 0 }),
  pushSessionToGitHub: vi.fn().mockResolvedValue(undefined),
  abandonSessionChanges: vi.fn().mockResolvedValue(undefined),
  getGitMergeStatus: vi.fn().mockResolvedValue({ status: 'clean' }),
  gitCreateBranch: vi.fn().mockResolvedValue(undefined),
  gitListBranches: vi.fn().mockResolvedValue(['main', 'dev']),
  gitStatus: vi.fn().mockResolvedValue({ clean: true }),
  configureGitRemote: vi.fn().mockResolvedValue(undefined),
}));

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return ({ children }: { children: React.ReactNode }) =>
    React.createElement(QueryClientProvider, { client: queryClient }, children);
}

describe('workspaceKeys', () => {
  it('generates correct query keys', () => {
    expect(workspaceKeys.all).toEqual(['workspace']);
    expect(workspaceKeys.list('proj', 'sess')).toEqual(['workspace', 'list', 'proj', 'sess', undefined]);
    expect(workspaceKeys.list('proj', 'sess', '/src')).toEqual(['workspace', 'list', 'proj', 'sess', '/src']);
    expect(workspaceKeys.file('proj', 'sess', 'f.txt')).toEqual(['workspace', 'file', 'proj', 'sess', 'f.txt']);
    expect(workspaceKeys.diff('proj', 'sess', 0)).toEqual(['workspace', 'diff', 'proj', 'sess', 0]);
  });
});

describe('useWorkspaceList', () => {
  it('fetches workspace listing', async () => {
    const wrapper = createWrapper();
    const { result } = renderHook(() => useWorkspaceList('proj', 'sess'), { wrapper });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toEqual([{ name: 'file.txt', type: 'file' }]);
  });

  it('is disabled when projectName is empty', () => {
    const wrapper = createWrapper();
    const { result } = renderHook(() => useWorkspaceList('', 'sess'), { wrapper });
    expect(result.current.fetchStatus).toBe('idle');
  });

  it('is disabled when sessionName is empty', () => {
    const wrapper = createWrapper();
    const { result } = renderHook(() => useWorkspaceList('proj', ''), { wrapper });
    expect(result.current.fetchStatus).toBe('idle');
  });

  it('respects enabled option', () => {
    const wrapper = createWrapper();
    const { result } = renderHook(() => useWorkspaceList('proj', 'sess', undefined, { enabled: false }), { wrapper });
    expect(result.current.fetchStatus).toBe('idle');
  });
});

describe('useWorkspaceFile', () => {
  it('fetches workspace file', async () => {
    const wrapper = createWrapper();
    const { result } = renderHook(() => useWorkspaceFile('proj', 'sess', 'file.txt'), { wrapper });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toBe('file content');
  });

  it('is disabled when path is empty', () => {
    const wrapper = createWrapper();
    const { result } = renderHook(() => useWorkspaceFile('proj', 'sess', ''), { wrapper });
    expect(result.current.fetchStatus).toBe('idle');
  });
});

describe('useWriteWorkspaceFile', () => {
  it('can be called as a mutation', async () => {
    const wrapper = createWrapper();
    const { result } = renderHook(() => useWriteWorkspaceFile(), { wrapper });
    expect(result.current.mutateAsync).toBeDefined();

    act(() => {
      result.current.mutate({
        projectName: 'proj',
        sessionName: 'sess',
        path: 'file.txt',
        content: 'new content',
      });
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
  });
});

describe('useSessionGitHubDiff', () => {
  it('fetches diff data', async () => {
    const wrapper = createWrapper();
    const { result } = renderHook(
      () => useSessionGitHubDiff('proj', 'sess', 0, '/repos/myrepo'),
      { wrapper },
    );
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toEqual({ files: { added: 1, removed: 0 }, total_added: 5, total_removed: 0 });
  });
});

describe('useAllSessionGitHubDiffs', () => {
  it('returns empty object when repos is empty', async () => {
    const wrapper = createWrapper();
    const { result } = renderHook(
      () => useAllSessionGitHubDiffs('proj', 'sess', [], (url) => url.split('/').pop()!),
      { wrapper },
    );
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toEqual({});
  });

  it('is disabled when repos is undefined', () => {
    const wrapper = createWrapper();
    const { result } = renderHook(
      () => useAllSessionGitHubDiffs('proj', 'sess', undefined, (url) => url.split('/').pop()!),
      { wrapper },
    );
    expect(result.current.fetchStatus).toBe('idle');
  });
});

describe('usePushSessionToGitHub', () => {
  it('can be called as a mutation', async () => {
    const wrapper = createWrapper();
    const { result } = renderHook(() => usePushSessionToGitHub(), { wrapper });

    act(() => {
      result.current.mutate({
        projectName: 'proj',
        sessionName: 'sess',
        repoIndex: 0,
        repoPath: '/repos/myrepo',
      });
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
  });
});

describe('useAbandonSessionChanges', () => {
  it('can be called as a mutation', async () => {
    const wrapper = createWrapper();
    const { result } = renderHook(() => useAbandonSessionChanges(), { wrapper });

    act(() => {
      result.current.mutate({
        projectName: 'proj',
        sessionName: 'sess',
        repoIndex: 0,
        repoPath: '/repos/myrepo',
      });
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
  });
});

describe('useGitMergeStatus', () => {
  it('fetches merge status', async () => {
    const wrapper = createWrapper();
    const { result } = renderHook(() => useGitMergeStatus('proj', 'sess'), { wrapper });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toEqual({ status: 'clean' });
  });

  it('is disabled when enabled is false', () => {
    const wrapper = createWrapper();
    const { result } = renderHook(() => useGitMergeStatus('proj', 'sess', 'artifacts', 'main', false), { wrapper });
    expect(result.current.fetchStatus).toBe('idle');
  });
});

describe('useGitCreateBranch', () => {
  it('can be called as a mutation', async () => {
    const wrapper = createWrapper();
    const { result } = renderHook(() => useGitCreateBranch(), { wrapper });

    act(() => {
      result.current.mutate({
        projectName: 'proj',
        sessionName: 'sess',
        branchName: 'feature-branch',
      });
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
  });
});

describe('useGitListBranches', () => {
  it('fetches branches', async () => {
    const wrapper = createWrapper();
    const { result } = renderHook(() => useGitListBranches('proj', 'sess'), { wrapper });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toEqual(['main', 'dev']);
  });

  it('is disabled when enabled is false', () => {
    const wrapper = createWrapper();
    const { result } = renderHook(() => useGitListBranches('proj', 'sess', 'artifacts', false), { wrapper });
    expect(result.current.fetchStatus).toBe('idle');
  });
});

describe('useGitStatus', () => {
  it('fetches git status', async () => {
    const wrapper = createWrapper();
    const { result } = renderHook(() => useGitStatus('proj', 'sess', '/workspace'), { wrapper });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toEqual({ clean: true });
  });

  it('is disabled when path is empty', () => {
    const wrapper = createWrapper();
    const { result } = renderHook(() => useGitStatus('proj', 'sess', ''), { wrapper });
    expect(result.current.fetchStatus).toBe('idle');
  });
});

describe('useConfigureGitRemote', () => {
  it('can be called as a mutation', async () => {
    const wrapper = createWrapper();
    const { result } = renderHook(() => useConfigureGitRemote(), { wrapper });

    act(() => {
      result.current.mutate({
        projectName: 'proj',
        sessionName: 'sess',
        path: '/workspace',
        remoteUrl: 'https://github.com/org/repo.git',
      });
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
  });
});
