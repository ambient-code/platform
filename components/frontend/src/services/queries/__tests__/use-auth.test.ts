import { renderHook, waitFor } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { useCurrentUser, authKeys } from '../use-auth';
import type { AuthPort } from '../../ports/auth';
import { createWrapper, createTestQueryClient } from './test-utils';
import { BACKEND_VERSION } from '../query-keys';

function createFakeAuthPort(overrides?: Partial<AuthPort>): AuthPort {
  return {
    getCurrentUser: vi.fn().mockResolvedValue({
      username: 'testuser',
      email: 'test@example.com',
      groups: ['developers'],
    }),
    ...overrides,
  };
}

describe('authKeys', () => {
  it('includes BACKEND_VERSION prefix', () => {
    expect(authKeys.all[0]).toBe(BACKEND_VERSION);
    expect(authKeys.currentUser()).toEqual([BACKEND_VERSION, 'auth', 'currentUser']);
  });
});

describe('useCurrentUser', () => {
  it('calls port.getCurrentUser and returns user data', async () => {
    const fakePort = createFakeAuthPort();
    const { result } = renderHook(() => useCurrentUser(fakePort), { wrapper: createWrapper() });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.getCurrentUser).toHaveBeenCalledOnce();
    expect(result.current.data).toEqual({
      username: 'testuser',
      email: 'test@example.com',
      groups: ['developers'],
    });
  });

  it('uses the correct query key', async () => {
    const queryClient = createTestQueryClient();
    const fakePort = createFakeAuthPort();
    const { result } = renderHook(() => useCurrentUser(fakePort), {
      wrapper: createWrapper(queryClient),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    const cache = queryClient.getQueryCache().findAll();
    expect(cache).toHaveLength(1);
    expect(cache[0].queryKey).toEqual(authKeys.currentUser());
  });

  it('propagates errors from the port', async () => {
    const fakePort = createFakeAuthPort({
      getCurrentUser: vi.fn().mockRejectedValue(new Error('Unauthorized')),
    });
    const { result } = renderHook(() => useCurrentUser(fakePort), { wrapper: createWrapper() });

    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(result.current.error).toBeInstanceOf(Error);
  });
});
