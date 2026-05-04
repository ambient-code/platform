import { renderHook, waitFor, act } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import {
  useMCPServerStatus,
  useConnectMCPServer,
  useDisconnectMCPServer,
  mcpCredentialsKeys,
} from '../use-mcp-credentials';
import { integrationsKeys } from '../use-integrations';
import type { McpCredentialsPort } from '../../ports/mcp-credentials';
import { createWrapper, createTestQueryClient } from './test-utils';
import { BACKEND_VERSION } from '../query-keys';

function createFakeMcpCredentialsPort(overrides?: Partial<McpCredentialsPort>): McpCredentialsPort {
  return {
    getMCPServerStatus: vi.fn().mockResolvedValue({ connected: true, serverName: 'test-server' }),
    connectMCPServer: vi.fn().mockResolvedValue(undefined),
    disconnectMCPServer: vi.fn().mockResolvedValue(undefined),
    ...overrides,
  };
}

describe('mcpCredentialsKeys', () => {
  it('includes BACKEND_VERSION prefix', () => {
    expect(mcpCredentialsKeys.all[0]).toBe(BACKEND_VERSION);
    expect(mcpCredentialsKeys.status('my-server')).toEqual([BACKEND_VERSION, 'mcp-credentials', 'my-server', 'status']);
  });
});

describe('useMCPServerStatus', () => {
  it('calls port.getMCPServerStatus with serverName and returns data', async () => {
    const fakePort = createFakeMcpCredentialsPort();
    const { result } = renderHook(() => useMCPServerStatus('test-server', fakePort), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.getMCPServerStatus).toHaveBeenCalledWith('test-server');
    expect(result.current.data).toEqual({ connected: true, serverName: 'test-server' });
  });

  it('uses the correct query key', async () => {
    const queryClient = createTestQueryClient();
    const fakePort = createFakeMcpCredentialsPort();
    const { result } = renderHook(() => useMCPServerStatus('test-server', fakePort), {
      wrapper: createWrapper(queryClient),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    const cache = queryClient.getQueryCache().findAll();
    expect(cache).toHaveLength(1);
    expect(cache[0].queryKey).toEqual(mcpCredentialsKeys.status('test-server'));
  });

  it('is disabled when serverName is empty', () => {
    const fakePort = createFakeMcpCredentialsPort();
    const { result } = renderHook(() => useMCPServerStatus('', fakePort), {
      wrapper: createWrapper(),
    });

    expect(result.current.fetchStatus).toBe('idle');
    expect(fakePort.getMCPServerStatus).not.toHaveBeenCalled();
  });

  it('propagates errors from the port', async () => {
    const fakePort = createFakeMcpCredentialsPort({
      getMCPServerStatus: vi.fn().mockRejectedValue(new Error('Not found')),
    });
    const { result } = renderHook(() => useMCPServerStatus('test-server', fakePort), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(result.current.error).toBeInstanceOf(Error);
  });
});

describe('useConnectMCPServer', () => {
  it('calls port.connectMCPServer with serverName and data', async () => {
    const fakePort = createFakeMcpCredentialsPort();
    const { result } = renderHook(() => useConnectMCPServer('test-server', fakePort), {
      wrapper: createWrapper(),
    });

    const connectData = { fields: { token: 'abc123' } };
    act(() => {
      result.current.mutate(connectData);
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.connectMCPServer).toHaveBeenCalledWith('test-server', connectData);
  });

  it('invalidates mcp credentials status cache on success', async () => {
    const queryClient = createTestQueryClient();
    queryClient.setQueryData(mcpCredentialsKeys.status('test-server'), { connected: false });
    const fakePort = createFakeMcpCredentialsPort();

    const { result } = renderHook(() => useConnectMCPServer('test-server', fakePort), {
      wrapper: createWrapper(queryClient),
    });

    act(() => {
      result.current.mutate({ fields: { token: 'abc123' } });
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(queryClient.getQueryState(mcpCredentialsKeys.status('test-server'))?.isInvalidated).toBe(true);
  });

  it('invalidates integrations status cache on success', async () => {
    const queryClient = createTestQueryClient();
    queryClient.setQueryData(integrationsKeys.status(), { cached: true });
    const fakePort = createFakeMcpCredentialsPort();

    const { result } = renderHook(() => useConnectMCPServer('test-server', fakePort), {
      wrapper: createWrapper(queryClient),
    });

    act(() => {
      result.current.mutate({ fields: { token: 'abc123' } });
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(queryClient.getQueryState(integrationsKeys.status())?.isInvalidated).toBe(true);
  });
});

describe('useDisconnectMCPServer', () => {
  it('calls port.disconnectMCPServer with serverName', async () => {
    const fakePort = createFakeMcpCredentialsPort();
    const { result } = renderHook(() => useDisconnectMCPServer('test-server', fakePort), {
      wrapper: createWrapper(),
    });

    act(() => {
      result.current.mutate();
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.disconnectMCPServer).toHaveBeenCalledWith('test-server');
  });

  it('invalidates mcp credentials status cache on success', async () => {
    const queryClient = createTestQueryClient();
    queryClient.setQueryData(mcpCredentialsKeys.status('test-server'), { connected: true });
    const fakePort = createFakeMcpCredentialsPort();

    const { result } = renderHook(() => useDisconnectMCPServer('test-server', fakePort), {
      wrapper: createWrapper(queryClient),
    });

    act(() => {
      result.current.mutate();
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(queryClient.getQueryState(mcpCredentialsKeys.status('test-server'))?.isInvalidated).toBe(true);
  });

  it('invalidates integrations status cache on success', async () => {
    const queryClient = createTestQueryClient();
    queryClient.setQueryData(integrationsKeys.status(), { cached: true });
    const fakePort = createFakeMcpCredentialsPort();

    const { result } = renderHook(() => useDisconnectMCPServer('test-server', fakePort), {
      wrapper: createWrapper(queryClient),
    });

    act(() => {
      result.current.mutate();
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(queryClient.getQueryState(integrationsKeys.status())?.isInvalidated).toBe(true);
  });
});
