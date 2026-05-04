import { renderHook, waitFor, act } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { mcpKeys, useMcpStatus, useUpdateSessionMcpServers } from '../use-mcp';
import type { SessionMcpPort } from '../../ports/session-mcp';
import { createWrapper, createTestQueryClient } from './test-utils';
import { BACKEND_VERSION } from '../query-keys';

const mockMcpStatus = {
  servers: [
    { name: 'mcp-server-1', status: 'connected', tools: [{ name: 'tool-1' }] },
  ],
};
const mockSession = { metadata: { name: 'sess-1' }, spec: {}, status: { phase: 'Running' } };

function createFakeSessionMcpPort(overrides?: Partial<SessionMcpPort>): SessionMcpPort {
  return {
    getMcpStatus: vi.fn().mockResolvedValue(mockMcpStatus),
    updateSessionMcpServers: vi.fn().mockResolvedValue(mockSession),
    ...overrides,
  };
}

describe('mcpKeys', () => {
  it('includes BACKEND_VERSION prefix in all keys', () => {
    expect(mcpKeys.all[0]).toBe(BACKEND_VERSION);
    expect(mcpKeys.all).toEqual([BACKEND_VERSION, 'mcp']);
  });

  it('generates correct status key', () => {
    expect(mcpKeys.status('proj', 'sess')).toEqual([BACKEND_VERSION, 'mcp', 'status', 'proj', 'sess']);
  });
});

describe('useMcpStatus', () => {
  it('calls port.getMcpStatus and returns data', async () => {
    const fakePort = createFakeSessionMcpPort();
    const { result } = renderHook(() => useMcpStatus('proj', 'sess', true, fakePort), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.getMcpStatus).toHaveBeenCalledWith('proj', 'sess');
    expect(result.current.data).toEqual(mockMcpStatus);
  });

  it('is disabled when projectName is empty', () => {
    const fakePort = createFakeSessionMcpPort();
    const { result } = renderHook(() => useMcpStatus('', 'sess', true, fakePort), {
      wrapper: createWrapper(),
    });

    expect(result.current.fetchStatus).toBe('idle');
    expect(fakePort.getMcpStatus).not.toHaveBeenCalled();
  });

  it('is disabled when sessionName is empty', () => {
    const fakePort = createFakeSessionMcpPort();
    const { result } = renderHook(() => useMcpStatus('proj', '', true, fakePort), {
      wrapper: createWrapper(),
    });

    expect(result.current.fetchStatus).toBe('idle');
    expect(fakePort.getMcpStatus).not.toHaveBeenCalled();
  });

  it('is disabled when enabled is false', () => {
    const fakePort = createFakeSessionMcpPort();
    const { result } = renderHook(() => useMcpStatus('proj', 'sess', false, fakePort), {
      wrapper: createWrapper(),
    });

    expect(result.current.fetchStatus).toBe('idle');
    expect(fakePort.getMcpStatus).not.toHaveBeenCalled();
  });

  it('is disabled when both projectName and sessionName are empty', () => {
    const fakePort = createFakeSessionMcpPort();
    const { result } = renderHook(() => useMcpStatus('', '', true, fakePort), {
      wrapper: createWrapper(),
    });

    expect(result.current.fetchStatus).toBe('idle');
    expect(fakePort.getMcpStatus).not.toHaveBeenCalled();
  });

  it('uses the correct query key', async () => {
    const queryClient = createTestQueryClient();
    const fakePort = createFakeSessionMcpPort();
    const { result } = renderHook(() => useMcpStatus('proj', 'sess', true, fakePort), {
      wrapper: createWrapper(queryClient),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    const cache = queryClient.getQueryCache().findAll();
    expect(cache[0].queryKey).toEqual(mcpKeys.status('proj', 'sess'));
  });

  it('defaults enabled to true when not provided', async () => {
    const fakePort = createFakeSessionMcpPort();
    const { result } = renderHook(() => useMcpStatus('proj', 'sess', undefined, fakePort), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.getMcpStatus).toHaveBeenCalledWith('proj', 'sess');
  });

  it('propagates errors from the port', async () => {
    const fakePort = createFakeSessionMcpPort({
      getMcpStatus: vi.fn().mockRejectedValue(new Error('MCP unavailable')),
    });
    const { result } = renderHook(() => useMcpStatus('proj', 'sess', true, fakePort), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(result.current.error).toBeInstanceOf(Error);
  });
});

describe('useUpdateSessionMcpServers', () => {
  it('calls port.updateSessionMcpServers with correct args', async () => {
    const fakePort = createFakeSessionMcpPort();
    const { result } = renderHook(() => useUpdateSessionMcpServers('proj', 'sess', fakePort), {
      wrapper: createWrapper(),
    });

    const mcpConfig = { custom: { 'mcp-server-1': { command: 'node', args: ['server.js'] } } };
    act(() => {
      result.current.mutate(mcpConfig);
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.updateSessionMcpServers).toHaveBeenCalledWith('proj', 'sess', mcpConfig);
    expect(result.current.data).toEqual(mockSession);
  });

  it('invalidates mcpKeys.status cache on success', async () => {
    const queryClient = createTestQueryClient();
    const fakePort = createFakeSessionMcpPort();

    const statusKey = mcpKeys.status('proj', 'sess');
    queryClient.setQueryData(statusKey, mockMcpStatus);

    const { result } = renderHook(() => useUpdateSessionMcpServers('proj', 'sess', fakePort), {
      wrapper: createWrapper(queryClient),
    });

    const mcpConfig = { custom: { 'mcp-server-2': { command: 'python', args: ['server.py'] } } };
    act(() => {
      result.current.mutate(mcpConfig);
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(queryClient.getQueryState(statusKey)?.isInvalidated).toBe(true);
  });
});
