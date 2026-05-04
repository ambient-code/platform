import { renderHook, waitFor, act } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
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
import type { SessionsPort } from '../../ports/sessions';
import type { SessionReposPort } from '../../ports/session-repos';
import { createWrapper, createTestQueryClient } from './test-utils';
import { BACKEND_VERSION } from '../query-keys';

const mockSession = { metadata: { name: 'sess-1' }, spec: {}, status: { phase: 'Running' } };
const mockPaginatedSessions = {
  items: [mockSession],
  totalCount: 1,
  hasMore: false,
  nextPage: undefined,
};

function createFakeSessionsPort(overrides?: Partial<SessionsPort>): SessionsPort {
  return {
    listSessions: vi.fn().mockResolvedValue(mockPaginatedSessions),
    getSession: vi.fn().mockResolvedValue(mockSession),
    createSession: vi.fn().mockResolvedValue({ metadata: { name: 'new-sess' } }),
    stopSession: vi.fn().mockResolvedValue('stopped'),
    startSession: vi.fn().mockResolvedValue({ message: 'started' }),
    cloneSession: vi.fn().mockResolvedValue({ metadata: { name: 'cloned-sess' } }),
    deleteSession: vi.fn().mockResolvedValue(undefined),
    getSessionPodEvents: vi.fn().mockResolvedValue([{ type: 'Normal', reason: 'Pulled' }]),
    updateSessionDisplayName: vi.fn().mockResolvedValue(mockSession),
    getSessionExport: vi.fn().mockResolvedValue({ events: [], messages: [] }),
    switchSessionModel: vi.fn().mockResolvedValue(mockSession),
    saveToGoogleDrive: vi.fn().mockResolvedValue({ fileId: '123', webViewLink: 'https://drive.google.com/file/123' }),
    ...overrides,
  };
}

function createFakeSessionReposPort(overrides?: Partial<SessionReposPort>): SessionReposPort {
  return {
    getReposStatus: vi.fn().mockResolvedValue({ repos: [] }),
    ...overrides,
  };
}

describe('sessionKeys', () => {
  it('includes BACKEND_VERSION prefix', () => {
    expect(sessionKeys.all[0]).toBe(BACKEND_VERSION);
  });

  it('generates correct query keys', () => {
    expect(sessionKeys.all).toEqual(['v1', 'sessions']);
    expect(sessionKeys.lists()).toEqual(['v1', 'sessions', 'list']);
    expect(sessionKeys.list('proj')).toEqual(['v1', 'sessions', 'list', 'proj', {}]);
    expect(sessionKeys.detail('proj', 'sess')).toEqual(['v1', 'sessions', 'detail', 'proj', 'sess']);
    expect(sessionKeys.messages('proj', 'sess')).toEqual(['v1', 'sessions', 'detail', 'proj', 'sess', 'messages']);
    expect(sessionKeys.export('proj', 'sess')).toEqual(['v1', 'sessions', 'detail', 'proj', 'sess', 'export']);
    expect(sessionKeys.reposStatus('proj', 'sess')).toEqual(['v1', 'sessions', 'detail', 'proj', 'sess', 'repos-status']);
  });
});

describe('useSessions', () => {
  it('fetches sessions list', async () => {
    const fakePort = createFakeSessionsPort();
    const { result } = renderHook(() => useSessions('proj', fakePort), {
      wrapper: createWrapper(),
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.listSessions).toHaveBeenCalledWith('proj');
    expect(result.current.data).toHaveLength(1);
  });

  it('is disabled when projectName is empty', () => {
    const fakePort = createFakeSessionsPort();
    const { result } = renderHook(() => useSessions('', fakePort), {
      wrapper: createWrapper(),
    });
    expect(result.current.fetchStatus).toBe('idle');
    expect(fakePort.listSessions).not.toHaveBeenCalled();
  });
});

describe('useSessionsPaginated', () => {
  it('fetches paginated sessions', async () => {
    const fakePort = createFakeSessionsPort();
    const { result } = renderHook(
      () => useSessionsPaginated('proj', { limit: 10 }, fakePort),
      { wrapper: createWrapper() },
    );
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.listSessions).toHaveBeenCalledWith('proj', { limit: 10 });
    expect(result.current.data?.totalCount).toBe(1);
  });
});

describe('useSession', () => {
  it('fetches a single session', async () => {
    const fakePort = createFakeSessionsPort();
    const { result } = renderHook(() => useSession('proj', 'sess-1', fakePort), {
      wrapper: createWrapper(),
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.getSession).toHaveBeenCalledWith('proj', 'sess-1');
    expect(result.current.data?.metadata.name).toBe('sess-1');
  });

  it('is disabled when projectName or sessionName is empty', () => {
    const fakePort = createFakeSessionsPort();
    const { result } = renderHook(() => useSession('', 'sess', fakePort), {
      wrapper: createWrapper(),
    });
    expect(result.current.fetchStatus).toBe('idle');

    const { result: result2 } = renderHook(() => useSession('proj', '', fakePort), {
      wrapper: createWrapper(),
    });
    expect(result2.current.fetchStatus).toBe('idle');
  });

  it('uses the correct query key', async () => {
    const queryClient = createTestQueryClient();
    const fakePort = createFakeSessionsPort();
    const { result } = renderHook(() => useSession('proj', 'sess-1', fakePort), {
      wrapper: createWrapper(queryClient),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    const cache = queryClient.getQueryCache().findAll();
    expect(cache[0].queryKey).toEqual(sessionKeys.detail('proj', 'sess-1'));
  });
});

describe('useCreateSession', () => {
  it('creates a session and invalidates list cache', async () => {
    const queryClient = createTestQueryClient();
    const fakePort = createFakeSessionsPort();

    queryClient.setQueryData(sessionKeys.list('proj'), { items: [] });

    const { result } = renderHook(() => useCreateSession(fakePort), {
      wrapper: createWrapper(queryClient),
    });

    act(() => {
      result.current.mutate({
        projectName: 'proj',
        data: { prompt: 'Hello' } as Parameters<typeof result.current.mutate>[0]['data'],
      });
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.createSession).toHaveBeenCalled();
    expect(result.current.data?.metadata.name).toBe('new-sess');
    expect(queryClient.getQueryState(sessionKeys.list('proj'))?.isInvalidated).toBe(true);
  });
});

describe('useStopSession', () => {
  it('stops a session and invalidates caches', async () => {
    const queryClient = createTestQueryClient();
    const fakePort = createFakeSessionsPort();

    queryClient.setQueryData(sessionKeys.detail('proj', 'sess-1'), mockSession);
    queryClient.setQueryData(sessionKeys.list('proj'), { items: [mockSession] });

    const { result } = renderHook(() => useStopSession(fakePort), {
      wrapper: createWrapper(queryClient),
    });

    act(() => {
      result.current.mutate({ projectName: 'proj', sessionName: 'sess-1' });
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.stopSession).toHaveBeenCalledWith('proj', 'sess-1', undefined);
    expect(queryClient.getQueryState(sessionKeys.detail('proj', 'sess-1'))?.isInvalidated).toBe(true);
    expect(queryClient.getQueryState(sessionKeys.list('proj'))?.isInvalidated).toBe(true);
  });
});

describe('useStartSession', () => {
  it('starts a session and invalidates caches', async () => {
    const queryClient = createTestQueryClient();
    const fakePort = createFakeSessionsPort();

    queryClient.setQueryData(sessionKeys.detail('proj', 'sess-1'), mockSession);
    queryClient.setQueryData(sessionKeys.list('proj'), { items: [mockSession] });

    const { result } = renderHook(() => useStartSession(fakePort), {
      wrapper: createWrapper(queryClient),
    });

    act(() => {
      result.current.mutate({ projectName: 'proj', sessionName: 'sess-1' });
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.startSession).toHaveBeenCalledWith('proj', 'sess-1');
    expect(queryClient.getQueryState(sessionKeys.detail('proj', 'sess-1'))?.isInvalidated).toBe(true);
    expect(queryClient.getQueryState(sessionKeys.list('proj'))?.isInvalidated).toBe(true);
  });
});

describe('useContinueSession', () => {
  it('continues a session via startSession', async () => {
    const fakePort = createFakeSessionsPort();
    const { result } = renderHook(() => useContinueSession(fakePort), {
      wrapper: createWrapper(),
    });

    act(() => {
      result.current.mutate({ projectName: 'proj', parentSessionName: 'sess-1' });
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.startSession).toHaveBeenCalledWith('proj', 'sess-1');
  });
});

describe('useCloneSession', () => {
  it('clones a session', async () => {
    const fakePort = createFakeSessionsPort();
    const { result } = renderHook(() => useCloneSession(fakePort), {
      wrapper: createWrapper(),
    });

    act(() => {
      result.current.mutate({
        projectName: 'proj',
        sessionName: 'sess-1',
        data: { prompt: 'Continue from here', targetProject: 'proj', newSessionName: 'cloned' } as Parameters<typeof result.current.mutate>[0]['data'],
      });
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.cloneSession).toHaveBeenCalled();
    expect(result.current.data?.metadata.name).toBe('cloned-sess');
  });
});

describe('useDeleteSession', () => {
  it('deletes a session and invalidates list cache', async () => {
    const queryClient = createTestQueryClient();
    const fakePort = createFakeSessionsPort();

    queryClient.setQueryData(sessionKeys.detail('proj', 'sess-1'), mockSession);
    queryClient.setQueryData(sessionKeys.list('proj'), { items: [mockSession] });

    const { result } = renderHook(() => useDeleteSession(fakePort), {
      wrapper: createWrapper(queryClient),
    });

    act(() => {
      result.current.mutate({ projectName: 'proj', sessionName: 'sess-1' });
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.deleteSession).toHaveBeenCalledWith('proj', 'sess-1');
    expect(queryClient.getQueryState(sessionKeys.list('proj'))?.isInvalidated).toBe(true);
  });
});

describe('useUpdateSessionDisplayName', () => {
  it('updates session display name', async () => {
    const fakePort = createFakeSessionsPort();
    const { result } = renderHook(() => useUpdateSessionDisplayName(fakePort), {
      wrapper: createWrapper(),
    });

    act(() => {
      result.current.mutate({
        projectName: 'proj',
        sessionName: 'sess-1',
        displayName: 'New Name',
      });
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.updateSessionDisplayName).toHaveBeenCalledWith('proj', 'sess-1', 'New Name');
  });
});

describe('useSessionExport', () => {
  it('fetches session export', async () => {
    const fakePort = createFakeSessionsPort();
    const { result } = renderHook(
      () => useSessionExport('proj', 'sess-1', true, fakePort),
      { wrapper: createWrapper() },
    );
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.getSessionExport).toHaveBeenCalledWith('proj', 'sess-1');
    expect(result.current.data).toEqual({ events: [], messages: [] });
  });

  it('is disabled when enabled is false', () => {
    const fakePort = createFakeSessionsPort();
    const { result } = renderHook(
      () => useSessionExport('proj', 'sess-1', false, fakePort),
      { wrapper: createWrapper() },
    );
    expect(result.current.fetchStatus).toBe('idle');
    expect(fakePort.getSessionExport).not.toHaveBeenCalled();
  });
});

describe('useSessionPodEvents', () => {
  it('fetches pod events', async () => {
    const fakePort = createFakeSessionsPort();
    const { result } = renderHook(
      () => useSessionPodEvents('proj', 'sess-1', 3000, fakePort),
      { wrapper: createWrapper() },
    );
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.getSessionPodEvents).toHaveBeenCalledWith('proj', 'sess-1');
    expect(result.current.data).toHaveLength(1);
  });
});

describe('useReposStatus', () => {
  it('fetches repos status', async () => {
    const fakePort = createFakeSessionReposPort();
    const { result } = renderHook(
      () => useReposStatus('proj', 'sess-1', true, fakePort),
      { wrapper: createWrapper() },
    );
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.getReposStatus).toHaveBeenCalledWith('proj', 'sess-1');
    expect(result.current.data).toEqual({ repos: [] });
  });

  it('is disabled when enabled is false', () => {
    const fakePort = createFakeSessionReposPort();
    const { result } = renderHook(
      () => useReposStatus('proj', 'sess-1', false, fakePort),
      { wrapper: createWrapper() },
    );
    expect(result.current.fetchStatus).toBe('idle');
    expect(fakePort.getReposStatus).not.toHaveBeenCalled();
  });
});
