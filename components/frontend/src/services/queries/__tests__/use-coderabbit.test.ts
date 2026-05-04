import { renderHook, waitFor, act } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { useConnectCodeRabbit, useDisconnectCodeRabbit, coderabbitKeys } from '../use-coderabbit';
import { integrationsKeys } from '../use-integrations';
import type { CodeRabbitPort } from '../../ports/coderabbit';
import { createWrapper, createTestQueryClient } from './test-utils';
import { BACKEND_VERSION } from '../query-keys';

function createFakeCodeRabbitPort(overrides?: Partial<CodeRabbitPort>): CodeRabbitPort {
  return {
    getCodeRabbitStatus: vi.fn().mockResolvedValue({ connected: true }),
    connectCodeRabbit: vi.fn().mockResolvedValue(undefined),
    disconnectCodeRabbit: vi.fn().mockResolvedValue(undefined),
    ...overrides,
  };
}

describe('coderabbitKeys', () => {
  it('includes BACKEND_VERSION prefix', () => {
    expect(coderabbitKeys.all[0]).toBe(BACKEND_VERSION);
    expect(coderabbitKeys.status()).toEqual([BACKEND_VERSION, 'coderabbit', 'status']);
  });
});

describe('useConnectCodeRabbit', () => {
  it('calls port.connectCodeRabbit with provided data', async () => {
    const fakePort = createFakeCodeRabbitPort();
    const { result } = renderHook(() => useConnectCodeRabbit(fakePort), {
      wrapper: createWrapper(),
    });

    const connectData = { apiKey: 'test-key' };
    act(() => {
      result.current.mutate(connectData);
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.connectCodeRabbit).toHaveBeenCalled();
    expect(vi.mocked(fakePort.connectCodeRabbit).mock.calls[0][0]).toEqual(connectData);
  });

  it('invalidates coderabbit status cache on success', async () => {
    const queryClient = createTestQueryClient();
    queryClient.setQueryData(coderabbitKeys.status(), { connected: false });
    const fakePort = createFakeCodeRabbitPort();

    const { result } = renderHook(() => useConnectCodeRabbit(fakePort), {
      wrapper: createWrapper(queryClient),
    });

    act(() => {
      result.current.mutate({ apiKey: 'test-key' });
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(queryClient.getQueryState(coderabbitKeys.status())?.isInvalidated).toBe(true);
  });

  it('invalidates integrations status cache on success', async () => {
    const queryClient = createTestQueryClient();
    queryClient.setQueryData(integrationsKeys.status(), { cached: true });
    const fakePort = createFakeCodeRabbitPort();

    const { result } = renderHook(() => useConnectCodeRabbit(fakePort), {
      wrapper: createWrapper(queryClient),
    });

    act(() => {
      result.current.mutate({ apiKey: 'test-key' });
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(queryClient.getQueryState(integrationsKeys.status())?.isInvalidated).toBe(true);
  });
});

describe('useDisconnectCodeRabbit', () => {
  it('calls port.disconnectCodeRabbit', async () => {
    const fakePort = createFakeCodeRabbitPort();
    const { result } = renderHook(() => useDisconnectCodeRabbit(fakePort), {
      wrapper: createWrapper(),
    });

    act(() => {
      result.current.mutate();
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.disconnectCodeRabbit).toHaveBeenCalledOnce();
  });

  it('invalidates coderabbit status cache on success', async () => {
    const queryClient = createTestQueryClient();
    queryClient.setQueryData(coderabbitKeys.status(), { connected: true });
    const fakePort = createFakeCodeRabbitPort();

    const { result } = renderHook(() => useDisconnectCodeRabbit(fakePort), {
      wrapper: createWrapper(queryClient),
    });

    act(() => {
      result.current.mutate();
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(queryClient.getQueryState(coderabbitKeys.status())?.isInvalidated).toBe(true);
  });

  it('invalidates integrations status cache on success', async () => {
    const queryClient = createTestQueryClient();
    queryClient.setQueryData(integrationsKeys.status(), { cached: true });
    const fakePort = createFakeCodeRabbitPort();

    const { result } = renderHook(() => useDisconnectCodeRabbit(fakePort), {
      wrapper: createWrapper(queryClient),
    });

    act(() => {
      result.current.mutate();
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(queryClient.getQueryState(integrationsKeys.status())?.isInvalidated).toBe(true);
  });
});
