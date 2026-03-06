/* eslint-disable react/display-name */
import { renderHook, waitFor, act } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { describe, it, expect, vi } from 'vitest';
import React from 'react';
import {
  useSessions,
  useSessionsPaginated,
  useSession,
  useCreateSession,
  useStopSession,
  useStartSession,
  useCloneSession,
  useDeleteSession,
  useContinueSession,
  useUpdateSessionDisplayName,
  useSessionExport,
  useSessionPodEvents,
  useReposStatus,
  sessionKeys,
} from '../use-sessions';

vi.mock('@/services/api/sessions', () => ({
  listSessions: vi.fn().mockResolvedValue([
    { metadata: { name: 'sess-1' }, status: { phase: 'Running' } },
  ]),
  listSessionsPaginated: vi.fn().mockResolvedValue({
    items: [{ metadata: { name: 'sess-1' }, status: { phase: 'Running' } }],
    totalCount: 1,
  }),
  getSession: vi.fn().mockResolvedValue({ metadata: { name: 'sess-1' }, status: { phase: 'Running' } }),
  createSession: vi.fn().mockResolvedValue({ metadata: { name: 'new-sess' } }),
  stopSession: vi.fn().mockResolvedValue({ message: 'stopped' }),
  startSession: vi.fn().mockResolvedValue({ metadata: { name: 'sess-1' }, status: { phase: 'Creating' } }),
  cloneSession: vi.fn().mockResolvedValue({ metadata: { name: 'cloned-sess' } }),
  deleteSession: vi.fn().mockResolvedValue(undefined),
  getSessionPodEvents: vi.fn().mockResolvedValue([{ type: 'Normal', reason: 'Pulled' }]),
  updateSessionDisplayName: vi.fn().mockResolvedValue(undefined),
  getSessionExport: vi.fn().mockResolvedValue({ events: [], messages: [] }),
  getReposStatus: vi.fn().mockResolvedValue({ repos: [] }),
}));

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return ({ children }: { children: React.ReactNode }) =>
    React.createElement(QueryClientProvider, { client: queryClient }, children);
}

describe('sessionKeys', () => {
  it('generates correct query keys', () => {
    expect(sessionKeys.all).toEqual(['sessions']);
    expect(sessionKeys.lists()).toEqual(['sessions', 'list']);
    expect(sessionKeys.list('proj')).toEqual(['sessions', 'list', 'proj', {}]);
    expect(sessionKeys.detail('proj', 'sess')).toEqual(['sessions', 'detail', 'proj', 'sess']);
    expect(sessionKeys.messages('proj', 'sess')).toEqual(['sessions', 'detail', 'proj', 'sess', 'messages']);
    expect(sessionKeys.export('proj', 'sess')).toEqual(['sessions', 'detail', 'proj', 'sess', 'export']);
    expect(sessionKeys.reposStatus('proj', 'sess')).toEqual(['sessions', 'detail', 'proj', 'sess', 'repos-status']);
  });
});

describe('useSessions', () => {
  it('fetches sessions list', async () => {
    const wrapper = createWrapper();
    const { result } = renderHook(() => useSessions('proj'), { wrapper });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toHaveLength(1);
  });

  it('is disabled when projectName is empty', () => {
    const wrapper = createWrapper();
    const { result } = renderHook(() => useSessions(''), { wrapper });
    expect(result.current.fetchStatus).toBe('idle');
  });
});

describe('useSessionsPaginated', () => {
  it('fetches paginated sessions', async () => {
    const wrapper = createWrapper();
    const { result } = renderHook(() => useSessionsPaginated('proj', { limit: 10 }), { wrapper });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data?.totalCount).toBe(1);
  });
});

describe('useSession', () => {
  it('fetches a single session', async () => {
    const wrapper = createWrapper();
    const { result } = renderHook(() => useSession('proj', 'sess-1'), { wrapper });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data?.metadata.name).toBe('sess-1');
  });

  it('is disabled when projectName or sessionName is empty', () => {
    const wrapper = createWrapper();
    const { result } = renderHook(() => useSession('', 'sess'), { wrapper });
    expect(result.current.fetchStatus).toBe('idle');

    const { result: result2 } = renderHook(() => useSession('proj', ''), { wrapper });
    expect(result2.current.fetchStatus).toBe('idle');
  });
});

describe('useCreateSession', () => {
  it('creates a session', async () => {
    const wrapper = createWrapper();
    const { result } = renderHook(() => useCreateSession(), { wrapper });

    act(() => {
      result.current.mutate({
        projectName: 'proj',
        data: { prompt: 'Hello' } as Parameters<typeof result.current.mutate>[0]['data'],
      });
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data?.metadata.name).toBe('new-sess');
  });
});

describe('useStopSession', () => {
  it('stops a session', async () => {
    const wrapper = createWrapper();
    const { result } = renderHook(() => useStopSession(), { wrapper });

    act(() => {
      result.current.mutate({
        projectName: 'proj',
        sessionName: 'sess-1',
      });
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
  });
});

describe('useStartSession', () => {
  it('starts a session', async () => {
    const wrapper = createWrapper();
    const { result } = renderHook(() => useStartSession(), { wrapper });

    act(() => {
      result.current.mutate({
        projectName: 'proj',
        sessionName: 'sess-1',
      });
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
  });
});

describe('useContinueSession', () => {
  it('continues a session', async () => {
    const wrapper = createWrapper();
    const { result } = renderHook(() => useContinueSession(), { wrapper });

    act(() => {
      result.current.mutate({
        projectName: 'proj',
        parentSessionName: 'sess-1',
      });
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
  });
});

describe('useCloneSession', () => {
  it('clones a session', async () => {
    const wrapper = createWrapper();
    const { result } = renderHook(() => useCloneSession(), { wrapper });

    act(() => {
      result.current.mutate({
        projectName: 'proj',
        sessionName: 'sess-1',
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        data: { prompt: 'Continue from here' } as any,
      });
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data?.metadata.name).toBe('cloned-sess');
  });
});

describe('useDeleteSession', () => {
  it('deletes a session', async () => {
    const wrapper = createWrapper();
    const { result } = renderHook(() => useDeleteSession(), { wrapper });

    act(() => {
      result.current.mutate({
        projectName: 'proj',
        sessionName: 'sess-1',
      });
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
  });
});

describe('useUpdateSessionDisplayName', () => {
  it('updates session display name', async () => {
    const wrapper = createWrapper();
    const { result } = renderHook(() => useUpdateSessionDisplayName(), { wrapper });

    act(() => {
      result.current.mutate({
        projectName: 'proj',
        sessionName: 'sess-1',
        displayName: 'New Name',
      });
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
  });
});

describe('useSessionExport', () => {
  it('fetches session export', async () => {
    const wrapper = createWrapper();
    const { result } = renderHook(() => useSessionExport('proj', 'sess-1', true), { wrapper });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toEqual({ events: [], messages: [] });
  });

  it('is disabled when enabled is false', () => {
    const wrapper = createWrapper();
    const { result } = renderHook(() => useSessionExport('proj', 'sess-1', false), { wrapper });
    expect(result.current.fetchStatus).toBe('idle');
  });
});

describe('useSessionPodEvents', () => {
  it('fetches pod events', async () => {
    const wrapper = createWrapper();
    const { result } = renderHook(() => useSessionPodEvents('proj', 'sess-1'), { wrapper });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toHaveLength(1);
  });
});

describe('useReposStatus', () => {
  it('fetches repos status', async () => {
    const wrapper = createWrapper();
    const { result } = renderHook(() => useReposStatus('proj', 'sess-1'), { wrapper });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toEqual({ repos: [] });
  });

  it('is disabled when enabled is false', () => {
    const wrapper = createWrapper();
    const { result } = renderHook(() => useReposStatus('proj', 'sess-1', false), { wrapper });
    expect(result.current.fetchStatus).toBe('idle');
  });
});
