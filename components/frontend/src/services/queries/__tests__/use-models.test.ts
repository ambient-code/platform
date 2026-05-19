import { renderHook, waitFor } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { useModels, modelKeys } from '../use-models';
import type { ModelsPort } from '../../ports/models';
import { createWrapper, createTestQueryClient } from './test-utils';
import { BACKEND_VERSION } from '../query-keys';

const mockModelsResponse = {
  models: [
    { id: 'claude-sonnet-4', name: 'Claude Sonnet 4', provider: 'anthropic' },
    { id: 'claude-haiku-3', name: 'Claude Haiku 3', provider: 'anthropic' },
  ],
  defaultModel: 'claude-sonnet-4',
};

function createFakeModelsPort(overrides?: Partial<ModelsPort>): ModelsPort {
  return {
    getModelsForProject: vi.fn().mockResolvedValue(mockModelsResponse),
    ...overrides,
  };
}

describe('modelKeys', () => {
  it('includes BACKEND_VERSION prefix in forProject key', () => {
    const key = modelKeys.forProject('my-project');
    expect(key[0]).toBe(BACKEND_VERSION);
    expect(key).toEqual([BACKEND_VERSION, 'models', 'my-project']);
  });

  it('includes provider when specified', () => {
    expect(modelKeys.forProject('my-project', 'anthropic')).toEqual([
      BACKEND_VERSION,
      'models',
      'my-project',
      'anthropic',
    ]);
  });
});

describe('useModels', () => {
  it('calls port.getModelsForProject and returns data', async () => {
    const fakePort = createFakeModelsPort();
    const { result } = renderHook(
      () => useModels('my-project', true, undefined, fakePort),
      { wrapper: createWrapper() },
    );

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.getModelsForProject).toHaveBeenCalledWith(
      'my-project',
      undefined,
    );
    expect(result.current.data).toEqual(mockModelsResponse);
  });

  it('is disabled when projectName is empty', () => {
    const fakePort = createFakeModelsPort();
    const { result } = renderHook(
      () => useModels('', true, undefined, fakePort),
      { wrapper: createWrapper() },
    );

    expect(result.current.fetchStatus).toBe('idle');
    expect(fakePort.getModelsForProject).not.toHaveBeenCalled();
  });

  it('is disabled when enabled param is false', () => {
    const fakePort = createFakeModelsPort();
    const { result } = renderHook(
      () => useModels('my-project', false, undefined, fakePort),
      { wrapper: createWrapper() },
    );

    expect(result.current.fetchStatus).toBe('idle');
    expect(fakePort.getModelsForProject).not.toHaveBeenCalled();
  });

  it('uses the correct cache key', async () => {
    const queryClient = createTestQueryClient();
    const fakePort = createFakeModelsPort();
    const { result } = renderHook(
      () => useModels('my-project', true, undefined, fakePort),
      { wrapper: createWrapper(queryClient) },
    );

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    const cache = queryClient.getQueryCache().findAll();
    expect(cache).toHaveLength(1);
    expect(cache[0].queryKey).toEqual(modelKeys.forProject('my-project'));
  });

  it('passes provider to port and uses it in cache key', async () => {
    const queryClient = createTestQueryClient();
    const fakePort = createFakeModelsPort();
    const { result } = renderHook(
      () => useModels('my-project', true, 'anthropic', fakePort),
      { wrapper: createWrapper(queryClient) },
    );

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.getModelsForProject).toHaveBeenCalledWith(
      'my-project',
      'anthropic',
    );
    const cache = queryClient.getQueryCache().findAll();
    expect(cache[0].queryKey).toEqual(
      modelKeys.forProject('my-project', 'anthropic'),
    );
  });

  it('propagates errors from the port', async () => {
    const fakePort = createFakeModelsPort({
      getModelsForProject: vi
        .fn()
        .mockRejectedValue(new Error('Models unavailable')),
    });
    const { result } = renderHook(
      () => useModels('my-project', true, undefined, fakePort),
      { wrapper: createWrapper() },
    );

    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(result.current.error).toBeInstanceOf(Error);
  });
});
