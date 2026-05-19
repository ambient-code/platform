import { renderHook, waitFor } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { useCapabilities, capabilitiesKeys } from '../use-capabilities';
import type { SessionCapabilitiesPort } from '../../ports/session-capabilities';
import { createWrapper, createTestQueryClient } from './test-utils';
import { BACKEND_VERSION } from '../query-keys';

const mockCapabilities = {
  framework: 'claude-code',
  agent_features: ['tool-use', 'streaming'],
  platform_features: ['mcp'],
  file_system: true,
  mcp: true,
  tracing: null,
  session_persistence: true,
  model: 'claude-sonnet-4',
  session_id: 'sess-123',
};

function createFakeCapabilitiesPort(
  overrides?: Partial<SessionCapabilitiesPort>,
): SessionCapabilitiesPort {
  return {
    getCapabilities: vi.fn().mockResolvedValue(mockCapabilities),
    ...overrides,
  };
}

describe('capabilitiesKeys', () => {
  it('includes BACKEND_VERSION prefix', () => {
    expect(capabilitiesKeys.all[0]).toBe(BACKEND_VERSION);
    expect(capabilitiesKeys.all).toEqual([BACKEND_VERSION, 'capabilities']);
  });

  it('session key includes project and session names', () => {
    expect(capabilitiesKeys.session('my-project', 'my-session')).toEqual([
      BACKEND_VERSION,
      'capabilities',
      'my-project',
      'my-session',
    ]);
  });
});

describe('useCapabilities', () => {
  it('calls port.getCapabilities and returns data', async () => {
    const fakePort = createFakeCapabilitiesPort();
    const { result } = renderHook(
      () => useCapabilities('my-project', 'my-session', true, fakePort),
      { wrapper: createWrapper() },
    );

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.getCapabilities).toHaveBeenCalledWith('my-project', 'my-session');
    expect(result.current.data).toEqual(mockCapabilities);
  });

  it('is disabled when projectName is empty', () => {
    const fakePort = createFakeCapabilitiesPort();
    const { result } = renderHook(
      () => useCapabilities('', 'my-session', true, fakePort),
      { wrapper: createWrapper() },
    );

    expect(result.current.fetchStatus).toBe('idle');
    expect(fakePort.getCapabilities).not.toHaveBeenCalled();
  });

  it('is disabled when sessionName is empty', () => {
    const fakePort = createFakeCapabilitiesPort();
    const { result } = renderHook(
      () => useCapabilities('my-project', '', true, fakePort),
      { wrapper: createWrapper() },
    );

    expect(result.current.fetchStatus).toBe('idle');
    expect(fakePort.getCapabilities).not.toHaveBeenCalled();
  });

  it('is disabled when enabled param is false', () => {
    const fakePort = createFakeCapabilitiesPort();
    const { result } = renderHook(
      () => useCapabilities('my-project', 'my-session', false, fakePort),
      { wrapper: createWrapper() },
    );

    expect(result.current.fetchStatus).toBe('idle');
    expect(fakePort.getCapabilities).not.toHaveBeenCalled();
  });

  it('uses the correct cache key', async () => {
    const queryClient = createTestQueryClient();
    const fakePort = createFakeCapabilitiesPort();
    const { result } = renderHook(
      () => useCapabilities('my-project', 'my-session', true, fakePort),
      { wrapper: createWrapper(queryClient) },
    );

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    const cache = queryClient.getQueryCache().findAll();
    expect(cache).toHaveLength(1);
    expect(cache[0].queryKey).toEqual(
      capabilitiesKeys.session('my-project', 'my-session'),
    );
  });

});
