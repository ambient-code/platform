import { renderHook, waitFor } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { useClusterInfo, clusterKeys } from '../use-cluster';
import type { ClusterPort } from '../../ports/cluster';
import { createWrapper, createTestQueryClient } from './test-utils';
import { BACKEND_VERSION } from '../query-keys';

const mockClusterInfo = {
  isOpenShift: true,
  vertexEnabled: false,
};

function createFakeClusterPort(overrides?: Partial<ClusterPort>): ClusterPort {
  return {
    getClusterInfo: vi.fn().mockResolvedValue(mockClusterInfo),
    ...overrides,
  };
}

describe('clusterKeys', () => {
  it('includes BACKEND_VERSION prefix', () => {
    expect(clusterKeys.all[0]).toBe(BACKEND_VERSION);
    expect(clusterKeys.all).toEqual([BACKEND_VERSION, 'cluster']);
  });

  it('info key extends all', () => {
    expect(clusterKeys.info()).toEqual([BACKEND_VERSION, 'cluster', 'info']);
  });
});

describe('useClusterInfo', () => {
  it('calls port.getClusterInfo and returns data', async () => {
    const fakePort = createFakeClusterPort();
    const { result } = renderHook(() => useClusterInfo(fakePort), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.getClusterInfo).toHaveBeenCalledOnce();
    expect(result.current.data).toEqual(mockClusterInfo);
  });

  it('uses the correct cache key', async () => {
    const queryClient = createTestQueryClient();
    const fakePort = createFakeClusterPort();
    const { result } = renderHook(() => useClusterInfo(fakePort), {
      wrapper: createWrapper(queryClient),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    const cache = queryClient.getQueryCache().findAll();
    expect(cache).toHaveLength(1);
    expect(cache[0].queryKey).toEqual(clusterKeys.info());
  });

});
