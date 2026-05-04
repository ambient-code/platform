import { renderHook, waitFor, act } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { keysKeys, useKeys, useCreateKey, useDeleteKey } from '../use-keys';
import type { KeysPort } from '../../ports/keys';
import { createWrapper, createTestQueryClient } from './test-utils';
import { BACKEND_VERSION } from '../query-keys';

const mockKey = { id: 'key-1', name: 'My Key', projectName: 'proj', createdAt: '2024-01-01' };
const mockKeys = [mockKey];
const mockCreateResponse = { id: 'key-2', name: 'New Key', key: 'sk-abc123' };

function createFakeKeysPort(overrides?: Partial<KeysPort>): KeysPort {
  return {
    listKeys: vi.fn().mockResolvedValue(mockKeys),
    createKey: vi.fn().mockResolvedValue(mockCreateResponse),
    deleteKey: vi.fn().mockResolvedValue(undefined),
    ...overrides,
  };
}

describe('keysKeys', () => {
  it('includes BACKEND_VERSION prefix in all keys', () => {
    expect(keysKeys.all[0]).toBe(BACKEND_VERSION);
    expect(keysKeys.all).toEqual([BACKEND_VERSION, 'keys']);
  });

  it('generates correct lists key', () => {
    expect(keysKeys.lists()).toEqual([BACKEND_VERSION, 'keys', 'list']);
  });

  it('generates correct list key with project', () => {
    expect(keysKeys.list('my-project')).toEqual([BACKEND_VERSION, 'keys', 'list', 'my-project']);
  });
});

describe('useKeys', () => {
  it('calls port.listKeys and returns data', async () => {
    const fakePort = createFakeKeysPort();
    const { result } = renderHook(() => useKeys('proj', fakePort), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.listKeys).toHaveBeenCalledWith('proj');
    expect(result.current.data).toEqual(mockKeys);
  });

  it('is disabled when projectName is empty', () => {
    const fakePort = createFakeKeysPort();
    const { result } = renderHook(() => useKeys('', fakePort), {
      wrapper: createWrapper(),
    });

    expect(result.current.fetchStatus).toBe('idle');
    expect(fakePort.listKeys).not.toHaveBeenCalled();
  });

  it('uses the correct query key', async () => {
    const queryClient = createTestQueryClient();
    const fakePort = createFakeKeysPort();
    const { result } = renderHook(() => useKeys('proj', fakePort), {
      wrapper: createWrapper(queryClient),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    const cache = queryClient.getQueryCache().findAll();
    expect(cache[0].queryKey).toEqual(keysKeys.list('proj'));
  });

  it('propagates errors from the port', async () => {
    const fakePort = createFakeKeysPort({
      listKeys: vi.fn().mockRejectedValue(new Error('Network error')),
    });
    const { result } = renderHook(() => useKeys('proj', fakePort), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(result.current.error).toBeInstanceOf(Error);
  });
});

describe('useCreateKey', () => {
  it('calls port.createKey with correct args', async () => {
    const fakePort = createFakeKeysPort();
    const { result } = renderHook(() => useCreateKey(fakePort), {
      wrapper: createWrapper(),
    });

    const createData = { name: 'New Key' };
    act(() => {
      result.current.mutate({ projectName: 'proj', data: createData });
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.createKey).toHaveBeenCalledWith('proj', createData);
    expect(result.current.data).toEqual(mockCreateResponse);
  });

  it('invalidates keysKeys.list cache on success', async () => {
    const queryClient = createTestQueryClient();
    const fakePort = createFakeKeysPort();

    const listKey = keysKeys.list('proj');
    queryClient.setQueryData(listKey, mockKeys);

    const { result } = renderHook(() => useCreateKey(fakePort), {
      wrapper: createWrapper(queryClient),
    });

    act(() => {
      result.current.mutate({ projectName: 'proj', data: { name: 'New Key' } });
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(queryClient.getQueryState(listKey)?.isInvalidated).toBe(true);
  });
});

describe('useDeleteKey', () => {
  it('calls port.deleteKey with correct args', async () => {
    const fakePort = createFakeKeysPort();
    const { result } = renderHook(() => useDeleteKey(fakePort), {
      wrapper: createWrapper(),
    });

    act(() => {
      result.current.mutate({ projectName: 'proj', keyId: 'key-1' });
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.deleteKey).toHaveBeenCalledWith('proj', 'key-1');
  });

  it('invalidates keysKeys.list cache on success', async () => {
    const queryClient = createTestQueryClient();
    const fakePort = createFakeKeysPort();

    const listKey = keysKeys.list('proj');
    queryClient.setQueryData(listKey, mockKeys);

    const { result } = renderHook(() => useDeleteKey(fakePort), {
      wrapper: createWrapper(queryClient),
    });

    act(() => {
      result.current.mutate({ projectName: 'proj', keyId: 'key-1' });
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(queryClient.getQueryState(listKey)?.isInvalidated).toBe(true);
  });
});
