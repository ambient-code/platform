import { renderHook, waitFor } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { versionKeys, useVersion } from '../use-version';
import type { VersionPort } from '../../ports/version';
import { createWrapper, createTestQueryClient } from './test-utils';
import { BACKEND_VERSION } from '../query-keys';

function createFakeVersionPort(overrides?: Partial<VersionPort>): VersionPort {
  return {
    getVersion: vi.fn().mockResolvedValue('1.2.3'),
    ...overrides,
  };
}

describe('versionKeys', () => {
  it('includes BACKEND_VERSION prefix', () => {
    expect(versionKeys.all[0]).toBe(BACKEND_VERSION);
  });

  it('generates correct query keys', () => {
    expect(versionKeys.all).toEqual([BACKEND_VERSION, 'version']);
    expect(versionKeys.current()).toEqual([BACKEND_VERSION, 'version', 'current']);
  });
});

describe('useVersion', () => {
  it('calls port.getVersion and returns version string', async () => {
    const fakePort = createFakeVersionPort();
    const { result } = renderHook(() => useVersion(fakePort), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.getVersion).toHaveBeenCalledOnce();
    expect(result.current.data).toBe('1.2.3');
  });

  it('uses the correct query key', async () => {
    const queryClient = createTestQueryClient();
    const fakePort = createFakeVersionPort();
    const { result } = renderHook(() => useVersion(fakePort), {
      wrapper: createWrapper(queryClient),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    const cache = queryClient.getQueryCache().findAll();
    expect(cache).toHaveLength(1);
    expect(cache[0].queryKey).toEqual(versionKeys.current());
  });

  it('is always enabled (no conditional params)', async () => {
    const fakePort = createFakeVersionPort();
    const { result } = renderHook(() => useVersion(fakePort), {
      wrapper: createWrapper(),
    });

    // Should immediately start fetching, not be idle
    expect(result.current.fetchStatus).toBe('fetching');
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
  });

  it('propagates errors from the port', async () => {
    const fakePort = createFakeVersionPort({
      getVersion: vi.fn().mockRejectedValue(new Error('Version unavailable')),
    });
    const { result } = renderHook(() => useVersion(fakePort), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(result.current.error).toBeInstanceOf(Error);
  });
});
