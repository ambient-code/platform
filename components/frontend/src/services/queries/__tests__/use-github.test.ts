import { renderHook, waitFor, act } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import {
  githubKeys,
  useGitHubStatus,
  useGitHubForks,
  usePRDiff,
  useConnectGitHub,
  useDisconnectGitHub,
  useCreateGitHubFork,
  useCreatePullRequest,
  useGitHubPATStatus,
  useSaveGitHubPAT,
  useDeleteGitHubPAT,
} from '../use-github';
import { integrationsKeys } from '../use-integrations';
import type { GitHubPort } from '../../ports/github';
import type { CreateForkRequest } from '@/types/api';
import { createWrapper, createTestQueryClient } from './test-utils';
import { BACKEND_VERSION } from '../query-keys';

function createFakeGitHubPort(overrides?: Partial<GitHubPort>): GitHubPort {
  return {
    getGitHubStatus: vi.fn().mockResolvedValue({
      connected: true,
      username: 'testuser',
    }),
    connectGitHub: vi.fn().mockResolvedValue('connected'),
    disconnectGitHub: vi.fn().mockResolvedValue('disconnected'),
    listGitHubForks: vi.fn().mockResolvedValue([
      { full_name: 'user/repo', owner: 'user', name: 'repo' },
    ]),
    createGitHubFork: vi.fn().mockResolvedValue({
      full_name: 'user/fork',
      owner: 'user',
      name: 'fork',
    }),
    getPRDiff: vi.fn().mockResolvedValue({
      files: [{ filename: 'test.ts', status: 'modified' }],
    }),
    createPullRequest: vi.fn().mockResolvedValue({
      url: 'https://github.com/org/repo/pull/1',
      number: 1,
    }),
    saveGitHubPAT: vi.fn().mockResolvedValue(undefined),
    getGitHubPATStatus: vi.fn().mockResolvedValue({
      configured: true,
      updatedAt: '2026-01-01T00:00:00Z',
    }),
    deleteGitHubPAT: vi.fn().mockResolvedValue(undefined),
    ...overrides,
  };
}

describe('githubKeys', () => {
  it('includes BACKEND_VERSION prefix', () => {
    expect(githubKeys.all[0]).toBe(BACKEND_VERSION);
  });

  it('generates correct query keys', () => {
    expect(githubKeys.all).toEqual([BACKEND_VERSION, 'github']);
    expect(githubKeys.status()).toEqual([BACKEND_VERSION, 'github', 'status']);
    expect(githubKeys.forks()).toEqual([BACKEND_VERSION, 'github', 'forks']);
    expect(githubKeys.forksForProject('proj', 'upstream/repo')).toEqual([
      BACKEND_VERSION,
      'github',
      'forks',
      'proj',
      'upstream/repo',
    ]);
    expect(githubKeys.forksForProject('proj')).toEqual([
      BACKEND_VERSION,
      'github',
      'forks',
      'proj',
      undefined,
    ]);
    expect(githubKeys.diff('owner', 'repo', 42)).toEqual([
      BACKEND_VERSION,
      'github',
      'diff',
      'owner',
      'repo',
      42,
    ]);
  });
});

// --- Query hooks ---

describe('useGitHubStatus', () => {
  it('calls port.getGitHubStatus and returns data', async () => {
    const fakePort = createFakeGitHubPort();
    const { result } = renderHook(() => useGitHubStatus(fakePort), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.getGitHubStatus).toHaveBeenCalledOnce();
    expect(result.current.data).toEqual({
      connected: true,
      username: 'testuser',
    });
  });

  it('uses the correct query key', async () => {
    const queryClient = createTestQueryClient();
    const fakePort = createFakeGitHubPort();
    const { result } = renderHook(() => useGitHubStatus(fakePort), {
      wrapper: createWrapper(queryClient),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    const cache = queryClient.getQueryCache().findAll();
    expect(cache).toHaveLength(1);
    expect(cache[0].queryKey).toEqual(githubKeys.status());
  });
});

describe('useGitHubForks', () => {
  it('calls port.listGitHubForks and returns data', async () => {
    const fakePort = createFakeGitHubPort();
    const { result } = renderHook(
      () => useGitHubForks('my-project', 'upstream/repo', fakePort),
      { wrapper: createWrapper() },
    );

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.listGitHubForks).toHaveBeenCalledWith('my-project', 'upstream/repo');
    expect(result.current.data).toHaveLength(1);
  });

  it('is disabled when projectName is empty', () => {
    const fakePort = createFakeGitHubPort();
    const { result } = renderHook(
      () => useGitHubForks('', 'upstream/repo', fakePort),
      { wrapper: createWrapper() },
    );

    expect(result.current.fetchStatus).toBe('idle');
    expect(fakePort.listGitHubForks).not.toHaveBeenCalled();
  });

  it('is disabled when upstreamRepo is empty', () => {
    const fakePort = createFakeGitHubPort();
    const { result } = renderHook(
      () => useGitHubForks('my-project', '', fakePort),
      { wrapper: createWrapper() },
    );

    expect(result.current.fetchStatus).toBe('idle');
    expect(fakePort.listGitHubForks).not.toHaveBeenCalled();
  });

  it('is disabled when upstreamRepo is undefined', () => {
    const fakePort = createFakeGitHubPort();
    const { result } = renderHook(
      () => useGitHubForks('my-project', undefined, fakePort),
      { wrapper: createWrapper() },
    );

    expect(result.current.fetchStatus).toBe('idle');
    expect(fakePort.listGitHubForks).not.toHaveBeenCalled();
  });
});

describe('usePRDiff', () => {
  it('calls port.getPRDiff and returns data', async () => {
    const fakePort = createFakeGitHubPort();
    const { result } = renderHook(
      () => usePRDiff('owner', 'repo', 42, 'proj', fakePort),
      { wrapper: createWrapper() },
    );

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.getPRDiff).toHaveBeenCalledWith('owner', 'repo', 42, 'proj');
    expect(result.current.data).toEqual({
      files: [{ filename: 'test.ts', status: 'modified' }],
    });
  });

  it('is disabled when owner is empty', () => {
    const fakePort = createFakeGitHubPort();
    const { result } = renderHook(
      () => usePRDiff('', 'repo', 42, 'proj', fakePort),
      { wrapper: createWrapper() },
    );

    expect(result.current.fetchStatus).toBe('idle');
    expect(fakePort.getPRDiff).not.toHaveBeenCalled();
  });

  it('is disabled when repo is empty', () => {
    const fakePort = createFakeGitHubPort();
    const { result } = renderHook(
      () => usePRDiff('owner', '', 42, 'proj', fakePort),
      { wrapper: createWrapper() },
    );

    expect(result.current.fetchStatus).toBe('idle');
    expect(fakePort.getPRDiff).not.toHaveBeenCalled();
  });

  it('is disabled when prNumber is 0', () => {
    const fakePort = createFakeGitHubPort();
    const { result } = renderHook(
      () => usePRDiff('owner', 'repo', 0, 'proj', fakePort),
      { wrapper: createWrapper() },
    );

    expect(result.current.fetchStatus).toBe('idle');
    expect(fakePort.getPRDiff).not.toHaveBeenCalled();
  });
});

describe('useGitHubPATStatus', () => {
  it('calls port.getGitHubPATStatus and returns data', async () => {
    const fakePort = createFakeGitHubPort();
    const { result } = renderHook(() => useGitHubPATStatus(fakePort), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.getGitHubPATStatus).toHaveBeenCalledOnce();
    expect(result.current.data).toEqual({
      configured: true,
      updatedAt: '2026-01-01T00:00:00Z',
    });
  });

  it('uses the correct query key', async () => {
    const queryClient = createTestQueryClient();
    const fakePort = createFakeGitHubPort();
    const { result } = renderHook(() => useGitHubPATStatus(fakePort), {
      wrapper: createWrapper(queryClient),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    const cache = queryClient.getQueryCache().findAll();
    expect(cache).toHaveLength(1);
    expect(cache[0].queryKey).toEqual([...githubKeys.all, 'pat', 'status']);
  });
});

// --- Mutation hooks ---

describe('useConnectGitHub', () => {
  it('calls port.connectGitHub with data', async () => {
    const fakePort = createFakeGitHubPort();
    const { result } = renderHook(() => useConnectGitHub(fakePort), {
      wrapper: createWrapper(),
    });

    act(() => {
      result.current.mutate({ installationId: 123 });
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.connectGitHub).toHaveBeenCalledWith({ installationId: 123 });
  });

  it('invalidates github status and integrations status on success', async () => {
    const queryClient = createTestQueryClient();
    const fakePort = createFakeGitHubPort();

    // Pre-seed caches
    queryClient.setQueryData(githubKeys.status(), { connected: false });
    queryClient.setQueryData(integrationsKeys.status(), { github: false });

    const { result } = renderHook(() => useConnectGitHub(fakePort), {
      wrapper: createWrapper(queryClient),
    });

    act(() => {
      result.current.mutate({ installationId: 123 });
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(
      queryClient.getQueryState(githubKeys.status())?.isInvalidated,
    ).toBe(true);
    expect(
      queryClient.getQueryState(integrationsKeys.status())?.isInvalidated,
    ).toBe(true);
  });
});

describe('useDisconnectGitHub', () => {
  it('calls port.disconnectGitHub', async () => {
    const fakePort = createFakeGitHubPort();
    const { result } = renderHook(() => useDisconnectGitHub(fakePort), {
      wrapper: createWrapper(),
    });

    act(() => {
      result.current.mutate();
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.disconnectGitHub).toHaveBeenCalledOnce();
  });

  it('invalidates github status, integrations status, and forks on success', async () => {
    const queryClient = createTestQueryClient();
    const fakePort = createFakeGitHubPort();

    // Pre-seed caches
    queryClient.setQueryData(githubKeys.status(), { connected: true });
    queryClient.setQueryData(integrationsKeys.status(), { github: true });
    queryClient.setQueryData(githubKeys.forks(), [{ full_name: 'user/repo' }]);

    const { result } = renderHook(() => useDisconnectGitHub(fakePort), {
      wrapper: createWrapper(queryClient),
    });

    act(() => {
      result.current.mutate();
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(
      queryClient.getQueryState(githubKeys.status())?.isInvalidated,
    ).toBe(true);
    expect(
      queryClient.getQueryState(integrationsKeys.status())?.isInvalidated,
    ).toBe(true);
    expect(
      queryClient.getQueryState(githubKeys.forks())?.isInvalidated,
    ).toBe(true);
  });
});

describe('useCreateGitHubFork', () => {
  it('calls port.createGitHubFork with data and projectName', async () => {
    const fakePort = createFakeGitHubPort();
    const { result } = renderHook(() => useCreateGitHubFork(fakePort), {
      wrapper: createWrapper(),
    });

    act(() => {
      result.current.mutate({
        data: { owner: 'org', repo: 'repo' } as CreateForkRequest,
        projectName: 'my-project',
      });
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.createGitHubFork).toHaveBeenCalledWith(
      { owner: 'org', repo: 'repo' },
      'my-project',
    );
  });

  it('invalidates forksForProject when projectName is provided', async () => {
    const queryClient = createTestQueryClient();
    const fakePort = createFakeGitHubPort();

    // Pre-seed cache for project-specific forks
    queryClient.setQueryData(githubKeys.forksForProject('my-project'), []);

    const { result } = renderHook(() => useCreateGitHubFork(fakePort), {
      wrapper: createWrapper(queryClient),
    });

    act(() => {
      result.current.mutate({
        data: { owner: 'org', repo: 'repo' } as CreateForkRequest,
        projectName: 'my-project',
      });
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(
      queryClient.getQueryState(githubKeys.forksForProject('my-project'))?.isInvalidated,
    ).toBe(true);
  });

  it('invalidates forks() when projectName is not provided', async () => {
    const queryClient = createTestQueryClient();
    const fakePort = createFakeGitHubPort();

    // Pre-seed generic forks cache
    queryClient.setQueryData(githubKeys.forks(), []);

    const { result } = renderHook(() => useCreateGitHubFork(fakePort), {
      wrapper: createWrapper(queryClient),
    });

    act(() => {
      result.current.mutate({
        data: { owner: 'org', repo: 'repo' } as CreateForkRequest,
      });
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(
      queryClient.getQueryState(githubKeys.forks())?.isInvalidated,
    ).toBe(true);
  });
});

describe('useCreatePullRequest', () => {
  it('calls port.createPullRequest with data and projectName', async () => {
    const fakePort = createFakeGitHubPort();
    const { result } = renderHook(() => useCreatePullRequest(fakePort), {
      wrapper: createWrapper(),
    });

    act(() => {
      result.current.mutate({
        data: {
          title: 'Test PR',
          head: 'feature',
          base: 'main',
        } as Parameters<typeof result.current.mutate>[0]['data'],
        projectName: 'my-project',
      });
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.createPullRequest).toHaveBeenCalledWith(
      { title: 'Test PR', head: 'feature', base: 'main' },
      'my-project',
    );
    expect(result.current.data).toEqual({
      url: 'https://github.com/org/repo/pull/1',
      number: 1,
    });
  });

  it('does not invalidate any cache', async () => {
    const queryClient = createTestQueryClient();
    const fakePort = createFakeGitHubPort();

    // Pre-seed some caches
    queryClient.setQueryData(githubKeys.status(), { connected: true });
    queryClient.setQueryData(githubKeys.forks(), []);

    const { result } = renderHook(() => useCreatePullRequest(fakePort), {
      wrapper: createWrapper(queryClient),
    });

    act(() => {
      result.current.mutate({
        data: {
          title: 'Test PR',
          head: 'feature',
          base: 'main',
        } as Parameters<typeof result.current.mutate>[0]['data'],
      });
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(
      queryClient.getQueryState(githubKeys.status())?.isInvalidated,
    ).toBe(false);
    expect(
      queryClient.getQueryState(githubKeys.forks())?.isInvalidated,
    ).toBe(false);
  });
});

describe('useSaveGitHubPAT', () => {
  it('calls port.saveGitHubPAT with token', async () => {
    const fakePort = createFakeGitHubPort();
    const { result } = renderHook(() => useSaveGitHubPAT(fakePort), {
      wrapper: createWrapper(),
    });

    act(() => {
      result.current.mutate('ghp_test_token_123');
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.saveGitHubPAT).toHaveBeenCalledWith('ghp_test_token_123');
  });

  it('invalidates PAT status, github status, and integrations status on success', async () => {
    const queryClient = createTestQueryClient();
    const fakePort = createFakeGitHubPort();

    const patStatusKey = [...githubKeys.all, 'pat', 'status'] as const;

    // Pre-seed caches
    queryClient.setQueryData(patStatusKey, { configured: false });
    queryClient.setQueryData(githubKeys.status(), { connected: false });
    queryClient.setQueryData(integrationsKeys.status(), { github: false });

    const { result } = renderHook(() => useSaveGitHubPAT(fakePort), {
      wrapper: createWrapper(queryClient),
    });

    act(() => {
      result.current.mutate('ghp_test_token_123');
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(
      queryClient.getQueryState(patStatusKey)?.isInvalidated,
    ).toBe(true);
    expect(
      queryClient.getQueryState(githubKeys.status())?.isInvalidated,
    ).toBe(true);
    expect(
      queryClient.getQueryState(integrationsKeys.status())?.isInvalidated,
    ).toBe(true);
  });
});

describe('useDeleteGitHubPAT', () => {
  it('calls port.deleteGitHubPAT', async () => {
    const fakePort = createFakeGitHubPort();
    const { result } = renderHook(() => useDeleteGitHubPAT(fakePort), {
      wrapper: createWrapper(),
    });

    act(() => {
      result.current.mutate();
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.deleteGitHubPAT).toHaveBeenCalledOnce();
  });

  it('invalidates PAT status, github status, and integrations status on success', async () => {
    const queryClient = createTestQueryClient();
    const fakePort = createFakeGitHubPort();

    const patStatusKey = [...githubKeys.all, 'pat', 'status'] as const;

    // Pre-seed caches
    queryClient.setQueryData(patStatusKey, { configured: true });
    queryClient.setQueryData(githubKeys.status(), { connected: true });
    queryClient.setQueryData(integrationsKeys.status(), { github: true });

    const { result } = renderHook(() => useDeleteGitHubPAT(fakePort), {
      wrapper: createWrapper(queryClient),
    });

    act(() => {
      result.current.mutate();
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(
      queryClient.getQueryState(patStatusKey)?.isInvalidated,
    ).toBe(true);
    expect(
      queryClient.getQueryState(githubKeys.status())?.isInvalidated,
    ).toBe(true);
    expect(
      queryClient.getQueryState(integrationsKeys.status())?.isInvalidated,
    ).toBe(true);
  });
});
