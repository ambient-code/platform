import { renderHook, waitFor, act } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { useJiraStatus, useConnectJira, useDisconnectJira, jiraKeys } from '../use-jira';
import type { JiraPort } from '../../ports/jira';
import { createWrapper, createTestQueryClient } from './test-utils';
import { BACKEND_VERSION } from '../query-keys';

function createFakeJiraPort(overrides?: Partial<JiraPort>): JiraPort {
  return {
    getJiraStatus: vi.fn().mockResolvedValue({ connected: true, baseUrl: 'https://jira.example.com' }),
    connectJira: vi.fn().mockResolvedValue(undefined),
    disconnectJira: vi.fn().mockResolvedValue(undefined),
    ...overrides,
  };
}

describe('jiraKeys', () => {
  it('includes BACKEND_VERSION prefix', () => {
    expect(jiraKeys.all[0]).toBe(BACKEND_VERSION);
    expect(jiraKeys.status()).toEqual([BACKEND_VERSION, 'jira', 'status']);
  });
});

describe('useJiraStatus', () => {
  it('calls port.getJiraStatus and returns data', async () => {
    const fakePort = createFakeJiraPort();
    const { result } = renderHook(() => useJiraStatus(fakePort), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.getJiraStatus).toHaveBeenCalledOnce();
    expect(result.current.data).toEqual({ connected: true, baseUrl: 'https://jira.example.com' });
  });

  it('uses the correct query key', async () => {
    const queryClient = createTestQueryClient();
    const fakePort = createFakeJiraPort();
    const { result } = renderHook(() => useJiraStatus(fakePort), {
      wrapper: createWrapper(queryClient),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    const cache = queryClient.getQueryCache().findAll();
    expect(cache).toHaveLength(1);
    expect(cache[0].queryKey).toEqual(jiraKeys.status());
  });

  it('propagates errors from the port', async () => {
    const fakePort = createFakeJiraPort({
      getJiraStatus: vi.fn().mockRejectedValue(new Error('Unauthorized')),
    });
    const { result } = renderHook(() => useJiraStatus(fakePort), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(result.current.error).toBeInstanceOf(Error);
  });
});

describe('useConnectJira', () => {
  it('calls port.connectJira with provided data', async () => {
    const fakePort = createFakeJiraPort();
    const { result } = renderHook(() => useConnectJira(fakePort), {
      wrapper: createWrapper(),
    });

    const connectData = { url: 'https://jira.example.com', email: 'user@example.com', apiToken: 'token-123' };
    act(() => {
      result.current.mutate(connectData);
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.connectJira).toHaveBeenCalled();
    expect(vi.mocked(fakePort.connectJira).mock.calls[0][0]).toEqual(connectData);
  });

  it('invalidates jira status cache on success', async () => {
    const queryClient = createTestQueryClient();
    queryClient.setQueryData(jiraKeys.status(), { connected: false });
    const fakePort = createFakeJiraPort();

    const { result } = renderHook(() => useConnectJira(fakePort), {
      wrapper: createWrapper(queryClient),
    });

    act(() => {
      result.current.mutate({ url: 'https://jira.example.com', email: 'user@example.com', apiToken: 'token-123' });
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(queryClient.getQueryState(jiraKeys.status())?.isInvalidated).toBe(true);
  });
});

describe('useDisconnectJira', () => {
  it('calls port.disconnectJira', async () => {
    const fakePort = createFakeJiraPort();
    const { result } = renderHook(() => useDisconnectJira(fakePort), {
      wrapper: createWrapper(),
    });

    act(() => {
      result.current.mutate();
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.disconnectJira).toHaveBeenCalledOnce();
  });

  it('invalidates jira status cache on success', async () => {
    const queryClient = createTestQueryClient();
    queryClient.setQueryData(jiraKeys.status(), { connected: true });
    const fakePort = createFakeJiraPort();

    const { result } = renderHook(() => useDisconnectJira(fakePort), {
      wrapper: createWrapper(queryClient),
    });

    act(() => {
      result.current.mutate();
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(queryClient.getQueryState(jiraKeys.status())?.isInvalidated).toBe(true);
  });
});
