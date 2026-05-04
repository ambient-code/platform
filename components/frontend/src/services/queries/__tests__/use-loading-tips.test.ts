import { renderHook, waitFor } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { useLoadingTips, configKeys } from '../use-loading-tips';
import type { ConfigPort } from '../../ports/config';
import { createWrapper, createTestQueryClient } from './test-utils';
import { BACKEND_VERSION } from '../query-keys';

const mockLoadingTips = {
  tips: ['Tip one', 'Tip two', 'Tip three'],
};

function createFakeConfigPort(overrides?: Partial<ConfigPort>): ConfigPort {
  return {
    getLoadingTips: vi.fn().mockResolvedValue(mockLoadingTips),
    ...overrides,
  };
}

describe('configKeys', () => {
  it('includes BACKEND_VERSION prefix', () => {
    expect(configKeys.all[0]).toBe(BACKEND_VERSION);
    expect(configKeys.all).toEqual([BACKEND_VERSION, 'config']);
  });

  it('loadingTips key extends all', () => {
    expect(configKeys.loadingTips()).toEqual([
      BACKEND_VERSION,
      'config',
      'loading-tips',
    ]);
  });
});

describe('useLoadingTips', () => {
  it('calls port.getLoadingTips and returns data', async () => {
    const fakePort = createFakeConfigPort();
    const { result } = renderHook(() => useLoadingTips(fakePort), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.getLoadingTips).toHaveBeenCalledOnce();
    expect(result.current.data).toEqual(mockLoadingTips);
  });

  it('uses the correct cache key', async () => {
    const queryClient = createTestQueryClient();
    const fakePort = createFakeConfigPort();
    const { result } = renderHook(() => useLoadingTips(fakePort), {
      wrapper: createWrapper(queryClient),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    const cache = queryClient.getQueryCache().findAll();
    expect(cache).toHaveLength(1);
    expect(cache[0].queryKey).toEqual(configKeys.loadingTips());
  });

});
