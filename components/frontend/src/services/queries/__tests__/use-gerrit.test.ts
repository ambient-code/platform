import { renderHook, waitFor, act } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import {
  useGerritInstances,
  useConnectGerrit,
  useDisconnectGerrit,
  useTestGerritConnection,
  gerritKeys,
} from '../use-gerrit';
import { integrationsKeys } from '../use-integrations';
import type { GerritPort } from '../../ports/gerrit';
import { createWrapper, createTestQueryClient } from './test-utils';
import { BACKEND_VERSION } from '../query-keys';

function createFakeGerritPort(overrides?: Partial<GerritPort>): GerritPort {
  return {
    getGerritInstances: vi.fn().mockResolvedValue({ instances: [] }),
    getGerritInstanceStatus: vi.fn().mockResolvedValue({ connected: true }),
    connectGerrit: vi.fn().mockResolvedValue(undefined),
    disconnectGerrit: vi.fn().mockResolvedValue(undefined),
    testGerritConnection: vi.fn().mockResolvedValue({ success: true }),
    ...overrides,
  };
}

describe('gerritKeys', () => {
  it('includes BACKEND_VERSION prefix', () => {
    expect(gerritKeys.all[0]).toBe(BACKEND_VERSION);
    expect(gerritKeys.instances()).toEqual([BACKEND_VERSION, 'gerrit', 'instances']);
  });
});

describe('useGerritInstances', () => {
  it('calls port.getGerritInstances and returns data', async () => {
    const fakePort = createFakeGerritPort();
    const { result } = renderHook(() => useGerritInstances(fakePort), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.getGerritInstances).toHaveBeenCalledOnce();
    expect(result.current.data).toEqual({ instances: [] });
  });

  it('uses the correct query key', async () => {
    const queryClient = createTestQueryClient();
    const fakePort = createFakeGerritPort();
    const { result } = renderHook(() => useGerritInstances(fakePort), {
      wrapper: createWrapper(queryClient),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    const cache = queryClient.getQueryCache().findAll();
    expect(cache).toHaveLength(1);
    expect(cache[0].queryKey).toEqual(gerritKeys.instances());
  });

  it('propagates errors from the port', async () => {
    const fakePort = createFakeGerritPort({
      getGerritInstances: vi.fn().mockRejectedValue(new Error('Network error')),
    });
    const { result } = renderHook(() => useGerritInstances(fakePort), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(result.current.error).toBeInstanceOf(Error);
  });
});

describe('useConnectGerrit', () => {
  it('calls port.connectGerrit with provided data', async () => {
    const fakePort = createFakeGerritPort();
    const { result } = renderHook(() => useConnectGerrit(fakePort), {
      wrapper: createWrapper(),
    });

    const connectData = { instanceName: 'my-gerrit', url: 'https://gerrit.example.com', authMethod: 'http_basic' as const, username: 'user', httpToken: 'token-123' };
    act(() => {
      result.current.mutate(connectData);
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.connectGerrit).toHaveBeenCalled();
    expect(vi.mocked(fakePort.connectGerrit).mock.calls[0][0]).toEqual(connectData);
  });

  it('invalidates integrations status cache on success', async () => {
    const queryClient = createTestQueryClient();
    queryClient.setQueryData(integrationsKeys.status(), { cached: true });
    const fakePort = createFakeGerritPort();

    const { result } = renderHook(() => useConnectGerrit(fakePort), {
      wrapper: createWrapper(queryClient),
    });

    act(() => {
      result.current.mutate({ instanceName: 'my-gerrit', url: 'https://gerrit.example.com', authMethod: 'http_basic' as const, username: 'user', httpToken: 'token-123' });
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(queryClient.getQueryState(integrationsKeys.status())?.isInvalidated).toBe(true);
  });

  it('invalidates gerrit instances cache on success', async () => {
    const queryClient = createTestQueryClient();
    queryClient.setQueryData(gerritKeys.instances(), { instances: [] });
    const fakePort = createFakeGerritPort();

    const { result } = renderHook(() => useConnectGerrit(fakePort), {
      wrapper: createWrapper(queryClient),
    });

    act(() => {
      result.current.mutate({ instanceName: 'my-gerrit', url: 'https://gerrit.example.com', authMethod: 'http_basic' as const, username: 'user', httpToken: 'token-123' });
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(queryClient.getQueryState(gerritKeys.instances())?.isInvalidated).toBe(true);
  });
});

describe('useDisconnectGerrit', () => {
  it('calls port.disconnectGerrit with instance name', async () => {
    const fakePort = createFakeGerritPort();
    const { result } = renderHook(() => useDisconnectGerrit(fakePort), {
      wrapper: createWrapper(),
    });

    act(() => {
      result.current.mutate('my-gerrit');
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.disconnectGerrit).toHaveBeenCalled();
    expect(vi.mocked(fakePort.disconnectGerrit).mock.calls[0][0]).toBe('my-gerrit');
  });

  it('invalidates integrations status cache on success', async () => {
    const queryClient = createTestQueryClient();
    queryClient.setQueryData(integrationsKeys.status(), { cached: true });
    const fakePort = createFakeGerritPort();

    const { result } = renderHook(() => useDisconnectGerrit(fakePort), {
      wrapper: createWrapper(queryClient),
    });

    act(() => {
      result.current.mutate('my-gerrit');
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(queryClient.getQueryState(integrationsKeys.status())?.isInvalidated).toBe(true);
  });

  it('invalidates gerrit instances cache on success', async () => {
    const queryClient = createTestQueryClient();
    queryClient.setQueryData(gerritKeys.instances(), { instances: [] });
    const fakePort = createFakeGerritPort();

    const { result } = renderHook(() => useDisconnectGerrit(fakePort), {
      wrapper: createWrapper(queryClient),
    });

    act(() => {
      result.current.mutate('my-gerrit');
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(queryClient.getQueryState(gerritKeys.instances())?.isInvalidated).toBe(true);
  });
});

describe('useTestGerritConnection', () => {
  it('calls port.testGerritConnection with provided data', async () => {
    const fakePort = createFakeGerritPort();
    const { result } = renderHook(() => useTestGerritConnection(fakePort), {
      wrapper: createWrapper(),
    });

    const testData = { url: 'https://gerrit.example.com', authMethod: 'http_basic' as const, username: 'user', httpToken: 'token-123' };
    act(() => {
      result.current.mutate(testData);
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.testGerritConnection).toHaveBeenCalled();
    expect(vi.mocked(fakePort.testGerritConnection).mock.calls[0][0]).toEqual(testData);
  });

  it('does not invalidate any cache on success', async () => {
    const queryClient = createTestQueryClient();
    queryClient.setQueryData(integrationsKeys.status(), { cached: true });
    queryClient.setQueryData(gerritKeys.instances(), { instances: [] });
    const fakePort = createFakeGerritPort();

    const { result } = renderHook(() => useTestGerritConnection(fakePort), {
      wrapper: createWrapper(queryClient),
    });

    act(() => {
      result.current.mutate({ url: 'https://gerrit.example.com', authMethod: 'http_basic' as const, username: 'user', httpToken: 'token-123' });
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(queryClient.getQueryState(integrationsKeys.status())?.isInvalidated).toBe(false);
    expect(queryClient.getQueryState(gerritKeys.instances())?.isInvalidated).toBe(false);
  });
});
