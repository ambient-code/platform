import { renderHook, waitFor, act } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { useGoogleStatus, useDisconnectGoogle, googleKeys } from '../use-google';
import type { GooglePort } from '../../ports/google';
import { createWrapper, createTestQueryClient } from './test-utils';
import { BACKEND_VERSION } from '../query-keys';

function createFakeGooglePort(overrides?: Partial<GooglePort>): GooglePort {
  return {
    getGoogleOAuthURL: vi.fn().mockResolvedValue({ url: 'https://accounts.google.com/oauth' }),
    getGoogleStatus: vi.fn().mockResolvedValue({ connected: true, email: 'user@gmail.com' }),
    disconnectGoogle: vi.fn().mockResolvedValue(undefined),
    ...overrides,
  };
}

describe('googleKeys', () => {
  it('includes BACKEND_VERSION prefix', () => {
    expect(googleKeys.all[0]).toBe(BACKEND_VERSION);
    expect(googleKeys.status()).toEqual([BACKEND_VERSION, 'google', 'status']);
  });
});

describe('useGoogleStatus', () => {
  it('calls port.getGoogleStatus and returns data', async () => {
    const fakePort = createFakeGooglePort();
    const { result } = renderHook(() => useGoogleStatus(fakePort), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.getGoogleStatus).toHaveBeenCalledOnce();
    expect(result.current.data).toEqual({ connected: true, email: 'user@gmail.com' });
  });

  it('uses the correct query key', async () => {
    const queryClient = createTestQueryClient();
    const fakePort = createFakeGooglePort();
    const { result } = renderHook(() => useGoogleStatus(fakePort), {
      wrapper: createWrapper(queryClient),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    const cache = queryClient.getQueryCache().findAll();
    expect(cache).toHaveLength(1);
    expect(cache[0].queryKey).toEqual(googleKeys.status());
  });

  it('propagates errors from the port', async () => {
    const fakePort = createFakeGooglePort({
      getGoogleStatus: vi.fn().mockRejectedValue(new Error('Unauthorized')),
    });
    const { result } = renderHook(() => useGoogleStatus(fakePort), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(result.current.error).toBeInstanceOf(Error);
  });
});

describe('useDisconnectGoogle', () => {
  it('calls port.disconnectGoogle', async () => {
    const fakePort = createFakeGooglePort();
    const { result } = renderHook(() => useDisconnectGoogle(fakePort), {
      wrapper: createWrapper(),
    });

    act(() => {
      result.current.mutate();
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.disconnectGoogle).toHaveBeenCalledOnce();
  });

  it('invalidates google status cache on success', async () => {
    const queryClient = createTestQueryClient();
    queryClient.setQueryData(googleKeys.status(), { connected: true, email: 'user@gmail.com' });
    const fakePort = createFakeGooglePort();

    const { result } = renderHook(() => useDisconnectGoogle(fakePort), {
      wrapper: createWrapper(queryClient),
    });

    act(() => {
      result.current.mutate();
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(queryClient.getQueryState(googleKeys.status())?.isInvalidated).toBe(true);
  });
});
