import { renderHook, waitFor, act } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { useGitLabStatus, useConnectGitLab, useDisconnectGitLab, gitlabKeys } from '../use-gitlab';
import type { GitLabPort } from '../../ports/gitlab';
import { createWrapper, createTestQueryClient } from './test-utils';
import { BACKEND_VERSION } from '../query-keys';

function createFakeGitLabPort(overrides?: Partial<GitLabPort>): GitLabPort {
  return {
    getGitLabStatus: vi.fn().mockResolvedValue({ connected: true, username: 'testuser' }),
    connectGitLab: vi.fn().mockResolvedValue(undefined),
    disconnectGitLab: vi.fn().mockResolvedValue(undefined),
    ...overrides,
  };
}

describe('gitlabKeys', () => {
  it('includes BACKEND_VERSION prefix', () => {
    expect(gitlabKeys.all[0]).toBe(BACKEND_VERSION);
    expect(gitlabKeys.status()).toEqual([BACKEND_VERSION, 'gitlab', 'status']);
  });
});

describe('useGitLabStatus', () => {
  it('calls port.getGitLabStatus and returns data', async () => {
    const fakePort = createFakeGitLabPort();
    const { result } = renderHook(() => useGitLabStatus(fakePort), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.getGitLabStatus).toHaveBeenCalledOnce();
    expect(result.current.data).toEqual({ connected: true, username: 'testuser' });
  });

  it('uses the correct query key', async () => {
    const queryClient = createTestQueryClient();
    const fakePort = createFakeGitLabPort();
    const { result } = renderHook(() => useGitLabStatus(fakePort), {
      wrapper: createWrapper(queryClient),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    const cache = queryClient.getQueryCache().findAll();
    expect(cache).toHaveLength(1);
    expect(cache[0].queryKey).toEqual(gitlabKeys.status());
  });

  it('propagates errors from the port', async () => {
    const fakePort = createFakeGitLabPort({
      getGitLabStatus: vi.fn().mockRejectedValue(new Error('Unauthorized')),
    });
    const { result } = renderHook(() => useGitLabStatus(fakePort), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(result.current.error).toBeInstanceOf(Error);
  });
});

describe('useConnectGitLab', () => {
  it('calls port.connectGitLab with provided data', async () => {
    const fakePort = createFakeGitLabPort();
    const { result } = renderHook(() => useConnectGitLab(fakePort), {
      wrapper: createWrapper(),
    });

    const connectData = { personalAccessToken: 'glpat-test-token' };
    act(() => {
      result.current.mutate(connectData);
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.connectGitLab).toHaveBeenCalled();
    expect(vi.mocked(fakePort.connectGitLab).mock.calls[0][0]).toEqual(connectData);
  });

  it('invalidates gitlab status cache on success', async () => {
    const queryClient = createTestQueryClient();
    queryClient.setQueryData(gitlabKeys.status(), { connected: false });
    const fakePort = createFakeGitLabPort();

    const { result } = renderHook(() => useConnectGitLab(fakePort), {
      wrapper: createWrapper(queryClient),
    });

    act(() => {
      result.current.mutate({ personalAccessToken: 'glpat-test-token' });
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(queryClient.getQueryState(gitlabKeys.status())?.isInvalidated).toBe(true);
  });
});

describe('useDisconnectGitLab', () => {
  it('calls port.disconnectGitLab', async () => {
    const fakePort = createFakeGitLabPort();
    const { result } = renderHook(() => useDisconnectGitLab(fakePort), {
      wrapper: createWrapper(),
    });

    act(() => {
      result.current.mutate();
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.disconnectGitLab).toHaveBeenCalledOnce();
  });

  it('invalidates gitlab status cache on success', async () => {
    const queryClient = createTestQueryClient();
    queryClient.setQueryData(gitlabKeys.status(), { connected: true });
    const fakePort = createFakeGitLabPort();

    const { result } = renderHook(() => useDisconnectGitLab(fakePort), {
      wrapper: createWrapper(queryClient),
    });

    act(() => {
      result.current.mutate();
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(queryClient.getQueryState(gitlabKeys.status())?.isInvalidated).toBe(true);
  });
});
