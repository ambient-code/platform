import { renderHook, waitFor, act } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { createSessionsAdapter } from '../../adapters/sessions';
import { createSessionReposAdapter } from '../../adapters/session-repos';
import { createSessionMcpAdapter } from '../../adapters/session-mcp';
import { createSessionCapabilitiesAdapter } from '../../adapters/session-capabilities';
import { createSessionTasksAdapter } from '../../adapters/session-tasks';
import {
  useSessionsPaginated,
  useSessions,
  useSession,
  useCreateSession,
  useStopSession,
  useStartSession,
  useCloneSession,
  useDeleteSession,
  useContinueSession,
  useUpdateSessionDisplayName,
  useSwitchSessionModel,
  useSessionExport,
  useSessionPodEvents,
  useReposStatus,
} from '../use-sessions';
import { useMcpStatus, useUpdateSessionMcpServers } from '../use-mcp';
import { useCapabilities } from '../use-capabilities';
import { createWrapper } from './test-utils';

const recordedSession = {
  metadata: { name: 'sess-1', namespace: 'proj-ns', annotations: {} },
  spec: { initialPrompt: 'hello', llmSettings: { model: 'claude-sonnet-4-20250514' }, timeout: 3600 },
  status: { phase: 'Running', agentStatus: 'idle' },
};

const recordedPaginatedResponse = {
  items: [recordedSession],
  totalCount: 1,
  hasMore: false,
  limit: 20,
  offset: 0,
};

describe('integration: hook → sessionsAdapter → fakeApi', () => {
  function createFakeSessionsApi() {
    return {
      listSessionsPaginated: vi.fn().mockResolvedValue(recordedPaginatedResponse),
      getSession: vi.fn().mockResolvedValue(recordedSession),
      createSession: vi.fn().mockResolvedValue(recordedSession),
      stopSession: vi.fn().mockResolvedValue('stopped'),
      startSession: vi.fn().mockResolvedValue({ message: 'started' }),
      cloneSession: vi.fn().mockResolvedValue(recordedSession),
      deleteSession: vi.fn().mockResolvedValue(undefined),
      getSessionPodEvents: vi.fn().mockResolvedValue([{ type: 'Normal', reason: 'Pulled', message: 'image pulled' }]),
      updateSessionDisplayName: vi.fn().mockResolvedValue(recordedSession),
      getSessionExport: vi.fn().mockResolvedValue({ sessionId: 'sess-1', projectName: 'proj', exportDate: '2026-01-01', aguiEvents: [{ type: 'message' }] }),
      switchSessionModel: vi.fn().mockResolvedValue(recordedSession),
      saveToGoogleDrive: vi.fn().mockResolvedValue({ fileId: 'abc', webViewLink: 'https://drive.google.com/abc' }),
      listSessions: vi.fn(),
      getReposStatus: vi.fn(),
      getMcpStatus: vi.fn(),
      updateSessionMcpServers: vi.fn(),
      getCapabilities: vi.fn(),
    };
  }

  it('useSessionsPaginated: API paginated response flows through adapter to hook', async () => {
    const fakeApi = createFakeSessionsApi();
    const adapter = createSessionsAdapter(fakeApi);

    const { result } = renderHook(
      () => useSessionsPaginated('proj', { limit: 20 }, adapter),
      { wrapper: createWrapper() },
    );

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.listSessionsPaginated).toHaveBeenCalledWith('proj', { limit: 20 });
    expect(result.current.data?.items).toHaveLength(1);
    expect(result.current.data?.items[0].metadata.name).toBe('sess-1');
    expect(result.current.data?.totalCount).toBe(1);
    expect(result.current.data?.hasMore).toBe(false);
    expect(result.current.data?.nextPage).toBeUndefined();
  });

  it('useSessionsPaginated: nextPage is defined when hasMore is true', async () => {
    const fakeApi = createFakeSessionsApi();
    fakeApi.listSessionsPaginated.mockResolvedValueOnce({
      ...recordedPaginatedResponse,
      hasMore: true,
    });
    const adapter = createSessionsAdapter(fakeApi);

    const { result } = renderHook(
      () => useSessionsPaginated('proj', {}, adapter),
      { wrapper: createWrapper() },
    );

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data?.hasMore).toBe(true);
    expect(result.current.data?.nextPage).toBeDefined();
  });

  it('useSessionsPaginated: nextPage() fetches second page with correct offset', async () => {
    const page1Session = {
      metadata: { name: 'sess-1', namespace: 'proj-ns', annotations: {} },
      spec: { initialPrompt: 'hello', llmSettings: { model: 'claude-sonnet-4-20250514' }, timeout: 3600 },
      status: { phase: 'Running', agentStatus: 'idle' },
    };
    const page2Session = {
      metadata: { name: 'sess-2', namespace: 'proj-ns', annotations: {} },
      spec: { initialPrompt: 'world', llmSettings: { model: 'claude-sonnet-4-20250514' }, timeout: 3600 },
      status: { phase: 'Stopped', agentStatus: 'idle' },
    };

    const fakeApi = createFakeSessionsApi();
    fakeApi.listSessionsPaginated
      .mockResolvedValueOnce({
        items: [page1Session],
        totalCount: 2,
        hasMore: true,
        limit: 1,
        offset: 0,
      })
      .mockResolvedValueOnce({
        items: [page2Session],
        totalCount: 2,
        hasMore: false,
        limit: 1,
        offset: 1,
      });
    const adapter = createSessionsAdapter(fakeApi);

    const { result } = renderHook(
      () => useSessionsPaginated('proj', { limit: 1 }, adapter),
      { wrapper: createWrapper() },
    );

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data?.items).toHaveLength(1);
    expect(result.current.data?.items[0].metadata.name).toBe('sess-1');
    expect(result.current.data?.nextPage).toBeDefined();

    const page2 = await result.current.data!.nextPage!();
    expect(fakeApi.listSessionsPaginated).toHaveBeenCalledTimes(2);
    expect(fakeApi.listSessionsPaginated.mock.calls[1][1]).toEqual(
      expect.objectContaining({ offset: 1, limit: 1 }),
    );
    expect(page2.items).toHaveLength(1);
    expect(page2.items[0].metadata.name).toBe('sess-2');
    expect(page2.hasMore).toBe(false);
    expect(page2.nextPage).toBeUndefined();
  });

  it('useSessions: returns items array from paginated response', async () => {
    const fakeApi = createFakeSessionsApi();
    const adapter = createSessionsAdapter(fakeApi);

    const { result } = renderHook(
      () => useSessions('proj', adapter),
      { wrapper: createWrapper() },
    );

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toHaveLength(1);
    expect(result.current.data?.[0].metadata.name).toBe('sess-1');
  });

  it('useSession: single session flows through', async () => {
    const fakeApi = createFakeSessionsApi();
    const adapter = createSessionsAdapter(fakeApi);

    const { result } = renderHook(
      () => useSession('proj', 'sess-1', adapter),
      { wrapper: createWrapper() },
    );

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.getSession).toHaveBeenCalledWith('proj', 'sess-1');
    expect(result.current.data?.status?.phase).toBe('Running');
  });

  it('useCreateSession: create flows through', async () => {
    const fakeApi = createFakeSessionsApi();
    const adapter = createSessionsAdapter(fakeApi);

    const { result } = renderHook(
      () => useCreateSession(adapter),
      { wrapper: createWrapper() },
    );

    act(() => {
      result.current.mutate({
        projectName: 'proj',
        data: { prompt: 'hello' } as Parameters<typeof result.current.mutate>[0]['data'],
      });
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.createSession).toHaveBeenCalledWith('proj', expect.objectContaining({ prompt: 'hello' }));
    expect(result.current.data?.metadata.name).toBe('sess-1');
  });

  it('useStopSession: stop flows through', async () => {
    const fakeApi = createFakeSessionsApi();
    const adapter = createSessionsAdapter(fakeApi);

    const { result } = renderHook(
      () => useStopSession(adapter),
      { wrapper: createWrapper() },
    );

    act(() => {
      result.current.mutate({ projectName: 'proj', sessionName: 'sess-1' });
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.stopSession).toHaveBeenCalledWith('proj', 'sess-1', undefined);
  });

  it('useDeleteSession: delete flows through', async () => {
    const fakeApi = createFakeSessionsApi();
    const adapter = createSessionsAdapter(fakeApi);

    const { result } = renderHook(
      () => useDeleteSession(adapter),
      { wrapper: createWrapper() },
    );

    act(() => {
      result.current.mutate({ projectName: 'proj', sessionName: 'sess-1' });
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.deleteSession).toHaveBeenCalledWith('proj', 'sess-1');
  });

  it('useSessionExport: export data flows through', async () => {
    const fakeApi = createFakeSessionsApi();
    const adapter = createSessionsAdapter(fakeApi);

    const { result } = renderHook(
      () => useSessionExport('proj', 'sess-1', true, adapter),
      { wrapper: createWrapper() },
    );

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.getSessionExport).toHaveBeenCalledWith('proj', 'sess-1');
    expect(result.current.data?.aguiEvents).toHaveLength(1);
    expect(result.current.data?.sessionId).toBe('sess-1');
  });

  it('useSessionPodEvents: pod events flow through', async () => {
    const fakeApi = createFakeSessionsApi();
    const adapter = createSessionsAdapter(fakeApi);

    const { result } = renderHook(
      () => useSessionPodEvents('proj', 'sess-1', 3000, adapter),
      { wrapper: createWrapper() },
    );

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.getSessionPodEvents).toHaveBeenCalledWith('proj', 'sess-1');
    expect(result.current.data).toHaveLength(1);
  });

  it('useStartSession: start flows through', async () => {
    const fakeApi = createFakeSessionsApi();
    const adapter = createSessionsAdapter(fakeApi);

    const { result } = renderHook(
      () => useStartSession(adapter),
      { wrapper: createWrapper() },
    );

    act(() => {
      result.current.mutate({ projectName: 'proj', sessionName: 'sess-1' });
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.startSession).toHaveBeenCalledWith('proj', 'sess-1');
  });

  it('useCloneSession: clone flows through', async () => {
    const fakeApi = createFakeSessionsApi();
    const adapter = createSessionsAdapter(fakeApi);

    const { result } = renderHook(
      () => useCloneSession(adapter),
      { wrapper: createWrapper() },
    );

    act(() => {
      result.current.mutate({
        projectName: 'proj',
        sessionName: 'sess-1',
        data: { targetProject: 'proj', newSessionName: 'sess-clone' },
      });
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.cloneSession).toHaveBeenCalledWith('proj', 'sess-1', { targetProject: 'proj', newSessionName: 'sess-clone' });
  });

  it('useContinueSession: continue flows through (delegates to startSession)', async () => {
    const fakeApi = createFakeSessionsApi();
    const adapter = createSessionsAdapter(fakeApi);

    const { result } = renderHook(
      () => useContinueSession(adapter),
      { wrapper: createWrapper() },
    );

    act(() => {
      result.current.mutate({ projectName: 'proj', parentSessionName: 'sess-1' });
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.startSession).toHaveBeenCalledWith('proj', 'sess-1');
  });

  it('useUpdateSessionDisplayName: display name update flows through', async () => {
    const fakeApi = createFakeSessionsApi();
    const adapter = createSessionsAdapter(fakeApi);

    const { result } = renderHook(
      () => useUpdateSessionDisplayName(adapter),
      { wrapper: createWrapper() },
    );

    act(() => {
      result.current.mutate({ projectName: 'proj', sessionName: 'sess-1', displayName: 'New Name' });
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.updateSessionDisplayName).toHaveBeenCalledWith('proj', 'sess-1', 'New Name');
  });

  it('useSwitchSessionModel: model switch flows through', async () => {
    const fakeApi = createFakeSessionsApi();
    const adapter = createSessionsAdapter(fakeApi);

    const { result } = renderHook(
      () => useSwitchSessionModel(adapter),
      { wrapper: createWrapper() },
    );

    act(() => {
      result.current.mutate({ projectName: 'proj', sessionName: 'sess-1', model: 'claude-opus-4-20250514' });
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.switchSessionModel).toHaveBeenCalledWith('proj', 'sess-1', 'claude-opus-4-20250514');
  });
});

describe('integration: hook → sessionReposAdapter → fakeApi', () => {
  it('useReposStatus: repos status flows through', async () => {
    const fakeApi = { getReposStatus: vi.fn().mockResolvedValue({ repos: [{ url: 'https://github.com/org/repo' }] }) };
    const adapter = createSessionReposAdapter(fakeApi);

    const { result } = renderHook(
      () => useReposStatus('proj', 'sess-1', true, adapter),
      { wrapper: createWrapper() },
    );

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.getReposStatus).toHaveBeenCalledWith('proj', 'sess-1');
    expect(result.current.data?.repos).toHaveLength(1);
  });
});

describe('integration: hook → sessionMcpAdapter → fakeApi', () => {
  it('useMcpStatus: MCP status flows through', async () => {
    const mcpResponse = { servers: [{ name: 'server-1', status: 'connected', tools: [] }] };
    const fakeApi = {
      getMcpStatus: vi.fn().mockResolvedValue(mcpResponse),
      updateSessionMcpServers: vi.fn(),
    };
    const adapter = createSessionMcpAdapter(fakeApi);

    const { result } = renderHook(
      () => useMcpStatus('proj', 'sess-1', true, adapter),
      { wrapper: createWrapper() },
    );

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.getMcpStatus).toHaveBeenCalledWith('proj', 'sess-1');
    expect(result.current.data?.servers).toHaveLength(1);
  });

  it('useUpdateSessionMcpServers: update flows through', async () => {
    const fakeApi = {
      getMcpStatus: vi.fn(),
      updateSessionMcpServers: vi.fn().mockResolvedValue(recordedSession),
    };
    const adapter = createSessionMcpAdapter(fakeApi);

    const { result } = renderHook(
      () => useUpdateSessionMcpServers('proj', 'sess-1', adapter),
      { wrapper: createWrapper() },
    );

    const config = { custom: { 'srv': { command: 'node', args: ['s.js'] } } };
    act(() => {
      result.current.mutate(config);
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.updateSessionMcpServers).toHaveBeenCalledWith('proj', 'sess-1', config);
  });
});

describe('integration: hook → sessionCapabilitiesAdapter → fakeApi', () => {
  it('useCapabilities: capabilities flow through', async () => {
    const capsResponse = { googleDrive: true, gitHub: true };
    const fakeApi = { getCapabilities: vi.fn().mockResolvedValue(capsResponse) };
    const adapter = createSessionCapabilitiesAdapter(fakeApi);

    const { result } = renderHook(
      () => useCapabilities('proj', 'sess-1', true, adapter),
      { wrapper: createWrapper() },
    );

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.getCapabilities).toHaveBeenCalledWith('proj', 'sess-1');
    expect(result.current.data).toEqual(capsResponse);
  });
});

describe('integration: hook → sessionTasksAdapter → fakeApi (method name mapping)', () => {
  it('adapter maps stopBackgroundTask → stopTask', async () => {
    const fakeApi = {
      stopBackgroundTask: vi.fn().mockResolvedValue(undefined),
      getTaskOutput: vi.fn().mockResolvedValue({ output: 'done' }),
    };
    const adapter = createSessionTasksAdapter(fakeApi);

    expect(adapter.stopTask).toBeDefined();
    await adapter.stopTask('proj', 'sess-1', 'task-1');
    expect(fakeApi.stopBackgroundTask).toHaveBeenCalledWith('proj', 'sess-1', 'task-1');
  });

  it('adapter maps getTaskOutput directly', async () => {
    const fakeApi = {
      stopBackgroundTask: vi.fn(),
      getTaskOutput: vi.fn().mockResolvedValue({ output: 'task result' }),
    };
    const adapter = createSessionTasksAdapter(fakeApi);

    const output = await adapter.getTaskOutput('proj', 'sess-1', 'task-1');
    expect(fakeApi.getTaskOutput).toHaveBeenCalledWith('proj', 'sess-1', 'task-1');
    expect(output).toEqual({ output: 'task result' });
  });
});
