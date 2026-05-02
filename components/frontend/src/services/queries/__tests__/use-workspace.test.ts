import { renderHook, waitFor, act } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
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
import type { SessionWorkspacePort } from '../../ports/session-workspace';
import { createWrapper, createTestQueryClient } from './test-utils';
import { BACKEND_VERSION } from '../query-keys';
import { sessionKeys } from '../use-sessions';

function createFakeWorkspacePort(overrides?: Partial<SessionWorkspacePort>): SessionWorkspacePort {
  return {
    listWorkspace: vi.fn().mockResolvedValue([{ name: 'file.txt', type: 'file' }]),
    readFile: vi.fn().mockResolvedValue('file content'),
    writeFile: vi.fn().mockResolvedValue(undefined),
    getGitHubDiff: vi.fn().mockResolvedValue({ files: { added: 1, removed: 0 }, total_added: 5, total_removed: 0 }),
    pushToGitHub: vi.fn().mockResolvedValue(undefined),
    abandonChanges: vi.fn().mockResolvedValue(undefined),
    getGitMergeStatus: vi.fn().mockResolvedValue({ status: 'clean' }),
    gitCreateBranch: vi.fn().mockResolvedValue(undefined),
    gitListBranches: vi.fn().mockResolvedValue(['main', 'dev']),
    gitStatus: vi.fn().mockResolvedValue({ clean: true }),
    configureGitRemote: vi.fn().mockResolvedValue(undefined),
    ...overrides,
  };
}

describe('workspaceKeys', () => {
  it('includes BACKEND_VERSION prefix', () => {
    expect(workspaceKeys.all[0]).toBe(BACKEND_VERSION);
  });

  it('generates correct query keys', () => {
    expect(workspaceKeys.all).toEqual(['v1', 'workspace']);
    expect(workspaceKeys.list('proj', 'sess')).toEqual(['v1', 'workspace', 'list', 'proj', 'sess', undefined]);
    expect(workspaceKeys.list('proj', 'sess', '/src')).toEqual(['v1', 'workspace', 'list', 'proj', 'sess', '/src']);
    expect(workspaceKeys.file('proj', 'sess', 'f.txt')).toEqual(['v1', 'workspace', 'file', 'proj', 'sess', 'f.txt']);
    expect(workspaceKeys.diff('proj', 'sess', 0)).toEqual(['v1', 'workspace', 'diff', 'proj', 'sess', 0]);
  });
});

describe('useWorkspaceList', () => {
  it('fetches workspace listing', async () => {
    const fakePort = createFakeWorkspacePort();
    const { result } = renderHook(
      () => useWorkspaceList('proj', 'sess', undefined, undefined, fakePort),
      { wrapper: createWrapper() },
    );
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.listWorkspace).toHaveBeenCalledWith('proj', 'sess', undefined);
    expect(result.current.data).toEqual([{ name: 'file.txt', type: 'file' }]);
  });

  it('is disabled when projectName is empty', () => {
    const fakePort = createFakeWorkspacePort();
    const { result } = renderHook(
      () => useWorkspaceList('', 'sess', undefined, undefined, fakePort),
      { wrapper: createWrapper() },
    );
    expect(result.current.fetchStatus).toBe('idle');
    expect(fakePort.listWorkspace).not.toHaveBeenCalled();
  });

  it('is disabled when sessionName is empty', () => {
    const fakePort = createFakeWorkspacePort();
    const { result } = renderHook(
      () => useWorkspaceList('proj', '', undefined, undefined, fakePort),
      { wrapper: createWrapper() },
    );
    expect(result.current.fetchStatus).toBe('idle');
    expect(fakePort.listWorkspace).not.toHaveBeenCalled();
  });

  it('respects enabled option', () => {
    const fakePort = createFakeWorkspacePort();
    const { result } = renderHook(
      () => useWorkspaceList('proj', 'sess', undefined, { enabled: false }, fakePort),
      { wrapper: createWrapper() },
    );
    expect(result.current.fetchStatus).toBe('idle');
    expect(fakePort.listWorkspace).not.toHaveBeenCalled();
  });
});

describe('useWorkspaceFile', () => {
  it('fetches workspace file', async () => {
    const fakePort = createFakeWorkspacePort();
    const { result } = renderHook(
      () => useWorkspaceFile('proj', 'sess', 'file.txt', undefined, fakePort),
      { wrapper: createWrapper() },
    );
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.readFile).toHaveBeenCalledWith('proj', 'sess', 'file.txt');
    expect(result.current.data).toBe('file content');
  });

  it('is disabled when path is empty', () => {
    const fakePort = createFakeWorkspacePort();
    const { result } = renderHook(
      () => useWorkspaceFile('proj', 'sess', '', undefined, fakePort),
      { wrapper: createWrapper() },
    );
    expect(result.current.fetchStatus).toBe('idle');
    expect(fakePort.readFile).not.toHaveBeenCalled();
  });
});

describe('useWriteWorkspaceFile', () => {
  it('writes a file and invalidates caches', async () => {
    const queryClient = createTestQueryClient();
    const fakePort = createFakeWorkspacePort();

    queryClient.setQueryData(workspaceKeys.file('proj', 'sess', 'file.txt'), 'old content');

    const { result } = renderHook(() => useWriteWorkspaceFile(fakePort), {
      wrapper: createWrapper(queryClient),
    });

    act(() => {
      result.current.mutate({
        projectName: 'proj',
        sessionName: 'sess',
        path: 'file.txt',
        content: 'new content',
      });
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.writeFile).toHaveBeenCalledWith('proj', 'sess', 'file.txt', 'new content');
    expect(queryClient.getQueryState(workspaceKeys.file('proj', 'sess', 'file.txt'))?.isInvalidated).toBe(true);
  });
});

describe('useSessionGitHubDiff', () => {
  it('fetches diff data', async () => {
    const fakePort = createFakeWorkspacePort();
    const { result } = renderHook(
      () => useSessionGitHubDiff('proj', 'sess', 0, '/repos/myrepo', undefined, fakePort),
      { wrapper: createWrapper() },
    );
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.getGitHubDiff).toHaveBeenCalledWith('proj', 'sess', 0, '/repos/myrepo');
    expect(result.current.data).toEqual({ files: { added: 1, removed: 0 }, total_added: 5, total_removed: 0 });
  });
});

describe('useAllSessionGitHubDiffs', () => {
  it('returns empty object when repos is empty', async () => {
    const fakePort = createFakeWorkspacePort();
    const { result } = renderHook(
      () => useAllSessionGitHubDiffs('proj', 'sess', [], (url) => url.split('/').pop()!, undefined, fakePort),
      { wrapper: createWrapper() },
    );
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toEqual({});
  });

  it('is disabled when repos is undefined', () => {
    const fakePort = createFakeWorkspacePort();
    const { result } = renderHook(
      () => useAllSessionGitHubDiffs('proj', 'sess', undefined, (url) => url.split('/').pop()!, undefined, fakePort),
      { wrapper: createWrapper() },
    );
    expect(result.current.fetchStatus).toBe('idle');
  });
});

describe('usePushSessionToGitHub', () => {
  it('pushes to GitHub and invalidates diff + session caches', async () => {
    const queryClient = createTestQueryClient();
    const fakePort = createFakeWorkspacePort();

    queryClient.setQueryData(workspaceKeys.diff('proj', 'sess', 0), { files: { added: 1, removed: 0 }, total_added: 5, total_removed: 0 });
    queryClient.setQueryData(sessionKeys.detail('proj', 'sess'), { metadata: { name: 'sess' } });

    const { result } = renderHook(() => usePushSessionToGitHub(fakePort), {
      wrapper: createWrapper(queryClient),
    });

    act(() => {
      result.current.mutate({
        projectName: 'proj',
        sessionName: 'sess',
        repoIndex: 0,
        repoPath: '/repos/myrepo',
      });
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.pushToGitHub).toHaveBeenCalledWith('proj', 'sess', 0, '/repos/myrepo');
    expect(queryClient.getQueryState(workspaceKeys.diff('proj', 'sess', 0))?.isInvalidated).toBe(true);
    expect(queryClient.getQueryState(sessionKeys.detail('proj', 'sess'))?.isInvalidated).toBe(true);
  });
});

describe('useAbandonSessionChanges', () => {
  it('abandons changes and invalidates caches', async () => {
    const queryClient = createTestQueryClient();
    const fakePort = createFakeWorkspacePort();

    queryClient.setQueryData(workspaceKeys.diff('proj', 'sess', 0), { files: { added: 1, removed: 0 }, total_added: 5, total_removed: 0 });

    const { result } = renderHook(() => useAbandonSessionChanges(fakePort), {
      wrapper: createWrapper(queryClient),
    });

    act(() => {
      result.current.mutate({
        projectName: 'proj',
        sessionName: 'sess',
        repoIndex: 0,
        repoPath: '/repos/myrepo',
      });
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.abandonChanges).toHaveBeenCalledWith('proj', 'sess', 0, '/repos/myrepo');
    expect(queryClient.getQueryState(workspaceKeys.diff('proj', 'sess', 0))?.isInvalidated).toBe(true);
  });
});

describe('useGitMergeStatus', () => {
  it('fetches merge status', async () => {
    const fakePort = createFakeWorkspacePort();
    const { result } = renderHook(
      () => useGitMergeStatus('proj', 'sess', 'artifacts', 'main', true, fakePort),
      { wrapper: createWrapper() },
    );
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.getGitMergeStatus).toHaveBeenCalledWith('proj', 'sess', 'artifacts', 'main');
    expect(result.current.data).toEqual({ status: 'clean' });
  });

  it('is disabled when enabled is false', () => {
    const fakePort = createFakeWorkspacePort();
    const { result } = renderHook(
      () => useGitMergeStatus('proj', 'sess', 'artifacts', 'main', false, fakePort),
      { wrapper: createWrapper() },
    );
    expect(result.current.fetchStatus).toBe('idle');
    expect(fakePort.getGitMergeStatus).not.toHaveBeenCalled();
  });
});

describe('useGitCreateBranch', () => {
  it('creates a branch', async () => {
    const fakePort = createFakeWorkspacePort();
    const { result } = renderHook(() => useGitCreateBranch(fakePort), {
      wrapper: createWrapper(),
    });

    act(() => {
      result.current.mutate({
        projectName: 'proj',
        sessionName: 'sess',
        branchName: 'feature-branch',
      });
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.gitCreateBranch).toHaveBeenCalledWith('proj', 'sess', 'feature-branch', 'artifacts');
  });
});

describe('useGitListBranches', () => {
  it('fetches branches', async () => {
    const fakePort = createFakeWorkspacePort();
    const { result } = renderHook(
      () => useGitListBranches('proj', 'sess', 'artifacts', true, fakePort),
      { wrapper: createWrapper() },
    );
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.gitListBranches).toHaveBeenCalledWith('proj', 'sess', 'artifacts');
    expect(result.current.data).toEqual(['main', 'dev']);
  });

  it('is disabled when enabled is false', () => {
    const fakePort = createFakeWorkspacePort();
    const { result } = renderHook(
      () => useGitListBranches('proj', 'sess', 'artifacts', false, fakePort),
      { wrapper: createWrapper() },
    );
    expect(result.current.fetchStatus).toBe('idle');
    expect(fakePort.gitListBranches).not.toHaveBeenCalled();
  });
});

describe('useGitStatus', () => {
  it('fetches git status', async () => {
    const fakePort = createFakeWorkspacePort();
    const { result } = renderHook(
      () => useGitStatus('proj', 'sess', '/workspace', undefined, fakePort),
      { wrapper: createWrapper() },
    );
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.gitStatus).toHaveBeenCalledWith('proj', 'sess', '/workspace');
    expect(result.current.data).toEqual({ clean: true });
  });

  it('is disabled when path is empty', () => {
    const fakePort = createFakeWorkspacePort();
    const { result } = renderHook(
      () => useGitStatus('proj', 'sess', '', undefined, fakePort),
      { wrapper: createWrapper() },
    );
    expect(result.current.fetchStatus).toBe('idle');
    expect(fakePort.gitStatus).not.toHaveBeenCalled();
  });
});

describe('useConfigureGitRemote', () => {
  it('configures git remote', async () => {
    const fakePort = createFakeWorkspacePort();
    const { result } = renderHook(() => useConfigureGitRemote(fakePort), {
      wrapper: createWrapper(),
    });

    act(() => {
      result.current.mutate({
        projectName: 'proj',
        sessionName: 'sess',
        path: '/workspace',
        remoteUrl: 'https://github.com/org/repo.git',
      });
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.configureGitRemote).toHaveBeenCalledWith('proj', 'sess', '/workspace', 'https://github.com/org/repo.git', 'main');
  });
});
