import { renderHook, waitFor, act } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import {
  scheduledSessionKeys,
  useScheduledSessions,
  useScheduledSession,
  useScheduledSessionRuns,
  useCreateScheduledSession,
  useUpdateScheduledSession,
  useDeleteScheduledSession,
  useSuspendScheduledSession,
  useResumeScheduledSession,
  useTriggerScheduledSession,
} from '../use-scheduled-sessions';
import type { ScheduledSessionsPort } from '../../ports/scheduled-sessions';
import { createWrapper, createTestQueryClient } from './test-utils';
import { BACKEND_VERSION } from '../query-keys';

const mockSchedule = {
  name: 'schedule-123',
  namespace: 'proj-ns',
  creationTimestamp: '2026-01-01T00:00:00Z',
  schedule: '0 9 * * *',
  suspend: false,
  displayName: 'Daily build',
  sessionTemplate: { initialPrompt: 'Run tests' },
  activeCount: 0,
};

function createFakeScheduledSessionsPort(
  overrides?: Partial<ScheduledSessionsPort>,
): ScheduledSessionsPort {
  return {
    listScheduledSessions: vi.fn().mockResolvedValue([mockSchedule]),
    getScheduledSession: vi.fn().mockResolvedValue({ ...mockSchedule, activeCount: 1 }),
    createScheduledSession: vi.fn().mockResolvedValue({
      ...mockSchedule,
      name: 'schedule-456',
      schedule: '0 12 * * 1',
      displayName: 'Weekly review',
    }),
    updateScheduledSession: vi.fn().mockResolvedValue({
      ...mockSchedule,
      schedule: '0 10 * * *',
      displayName: 'Updated schedule',
    }),
    deleteScheduledSession: vi.fn().mockResolvedValue(undefined),
    suspendScheduledSession: vi.fn().mockResolvedValue({ ...mockSchedule, suspend: true }),
    resumeScheduledSession: vi.fn().mockResolvedValue({ ...mockSchedule, suspend: false }),
    triggerScheduledSession: vi.fn().mockResolvedValue({ name: 'schedule-123-manual-abc', namespace: 'proj' }),
    listScheduledSessionRuns: vi.fn().mockResolvedValue([
      { metadata: { name: 'run-1' }, status: { phase: 'Running' } },
    ]),
    ...overrides,
  };
}

describe('scheduledSessionKeys', () => {
  it('includes BACKEND_VERSION prefix', () => {
    expect(scheduledSessionKeys.all[0]).toBe(BACKEND_VERSION);
  });

  it('generates correct query keys', () => {
    expect(scheduledSessionKeys.all).toEqual(['v1', 'scheduled-sessions']);
    expect(scheduledSessionKeys.lists()).toEqual(['v1', 'scheduled-sessions', 'list']);
    expect(scheduledSessionKeys.list('proj')).toEqual(['v1', 'scheduled-sessions', 'list', 'proj']);
    expect(scheduledSessionKeys.details()).toEqual(['v1', 'scheduled-sessions', 'detail']);
    expect(scheduledSessionKeys.detail('proj', 'sched-1')).toEqual([
      'v1', 'scheduled-sessions', 'detail', 'proj', 'sched-1',
    ]);
    expect(scheduledSessionKeys.runs('proj', 'sched-1')).toEqual([
      'v1', 'scheduled-sessions', 'detail', 'proj', 'sched-1', 'runs',
    ]);
  });
});

describe('useScheduledSessions', () => {
  it('fetches scheduled sessions list', async () => {
    const fakePort = createFakeScheduledSessionsPort();
    const { result } = renderHook(() => useScheduledSessions('proj', fakePort), {
      wrapper: createWrapper(),
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.listScheduledSessions).toHaveBeenCalledWith('proj');
    expect(result.current.data).toHaveLength(1);
    expect(result.current.data?.[0].name).toBe('schedule-123');
  });

  it('is disabled when projectName is empty', () => {
    const fakePort = createFakeScheduledSessionsPort();
    const { result } = renderHook(() => useScheduledSessions('', fakePort), {
      wrapper: createWrapper(),
    });
    expect(result.current.fetchStatus).toBe('idle');
    expect(fakePort.listScheduledSessions).not.toHaveBeenCalled();
  });
});

describe('useScheduledSession', () => {
  it('fetches a single scheduled session', async () => {
    const fakePort = createFakeScheduledSessionsPort();
    const { result } = renderHook(
      () => useScheduledSession('proj', 'schedule-123', fakePort),
      { wrapper: createWrapper() },
    );
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.getScheduledSession).toHaveBeenCalledWith('proj', 'schedule-123');
    expect(result.current.data?.name).toBe('schedule-123');
  });

  it('is disabled when projectName or name is empty', () => {
    const fakePort = createFakeScheduledSessionsPort();
    const { result } = renderHook(
      () => useScheduledSession('', 'schedule-123', fakePort),
      { wrapper: createWrapper() },
    );
    expect(result.current.fetchStatus).toBe('idle');

    const { result: result2 } = renderHook(
      () => useScheduledSession('proj', '', fakePort),
      { wrapper: createWrapper() },
    );
    expect(result2.current.fetchStatus).toBe('idle');
  });
});

describe('useScheduledSessionRuns', () => {
  it('fetches scheduled session runs', async () => {
    const fakePort = createFakeScheduledSessionsPort();
    const { result } = renderHook(
      () => useScheduledSessionRuns('proj', 'schedule-123', fakePort),
      { wrapper: createWrapper() },
    );
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.listScheduledSessionRuns).toHaveBeenCalledWith('proj', 'schedule-123');
    expect(result.current.data).toHaveLength(1);
    expect(result.current.data?.[0].metadata.name).toBe('run-1');
  });

  it('is disabled when projectName or name is empty', () => {
    const fakePort = createFakeScheduledSessionsPort();
    const { result } = renderHook(
      () => useScheduledSessionRuns('', 'schedule-123', fakePort),
      { wrapper: createWrapper() },
    );
    expect(result.current.fetchStatus).toBe('idle');

    const { result: result2 } = renderHook(
      () => useScheduledSessionRuns('proj', '', fakePort),
      { wrapper: createWrapper() },
    );
    expect(result2.current.fetchStatus).toBe('idle');
  });
});

describe('useCreateScheduledSession', () => {
  it('creates a scheduled session and invalidates list cache', async () => {
    const queryClient = createTestQueryClient();
    const fakePort = createFakeScheduledSessionsPort();

    queryClient.setQueryData(scheduledSessionKeys.list('proj'), []);

    const { result } = renderHook(() => useCreateScheduledSession(fakePort), {
      wrapper: createWrapper(queryClient),
    });

    act(() => {
      result.current.mutate({
        projectName: 'proj',
        data: {
          schedule: '0 12 * * 1',
          sessionTemplate: { initialPrompt: 'Review code' },
        } as Parameters<typeof result.current.mutate>[0]['data'],
      });
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.createScheduledSession).toHaveBeenCalled();
    expect(result.current.data?.name).toBe('schedule-456');
    expect(queryClient.getQueryState(scheduledSessionKeys.list('proj'))?.isInvalidated).toBe(true);
  });
});

describe('useUpdateScheduledSession', () => {
  it('updates a scheduled session and invalidates caches', async () => {
    const queryClient = createTestQueryClient();
    const fakePort = createFakeScheduledSessionsPort();

    queryClient.setQueryData(scheduledSessionKeys.detail('proj', 'schedule-123'), mockSchedule);
    queryClient.setQueryData(scheduledSessionKeys.list('proj'), [mockSchedule]);

    const { result } = renderHook(() => useUpdateScheduledSession(fakePort), {
      wrapper: createWrapper(queryClient),
    });

    act(() => {
      result.current.mutate({
        projectName: 'proj',
        name: 'schedule-123',
        data: { schedule: '0 10 * * *' },
      });
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.updateScheduledSession).toHaveBeenCalledWith('proj', 'schedule-123', { schedule: '0 10 * * *' });
    expect(result.current.data?.schedule).toBe('0 10 * * *');
    expect(queryClient.getQueryState(scheduledSessionKeys.detail('proj', 'schedule-123'))?.isInvalidated).toBe(true);
    expect(queryClient.getQueryState(scheduledSessionKeys.list('proj'))?.isInvalidated).toBe(true);
  });
});

describe('useDeleteScheduledSession', () => {
  it('deletes a scheduled session and invalidates list cache', async () => {
    const queryClient = createTestQueryClient();
    const fakePort = createFakeScheduledSessionsPort();

    queryClient.setQueryData(scheduledSessionKeys.list('proj'), [mockSchedule]);

    const { result } = renderHook(() => useDeleteScheduledSession(fakePort), {
      wrapper: createWrapper(queryClient),
    });

    act(() => {
      result.current.mutate({ projectName: 'proj', name: 'schedule-123' });
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.deleteScheduledSession).toHaveBeenCalledWith('proj', 'schedule-123');
    expect(queryClient.getQueryState(scheduledSessionKeys.list('proj'))?.isInvalidated).toBe(true);
  });
});

describe('useSuspendScheduledSession', () => {
  it('suspends a scheduled session and invalidates caches', async () => {
    const queryClient = createTestQueryClient();
    const fakePort = createFakeScheduledSessionsPort();

    queryClient.setQueryData(scheduledSessionKeys.detail('proj', 'schedule-123'), mockSchedule);
    queryClient.setQueryData(scheduledSessionKeys.list('proj'), [mockSchedule]);

    const { result } = renderHook(() => useSuspendScheduledSession(fakePort), {
      wrapper: createWrapper(queryClient),
    });

    act(() => {
      result.current.mutate({ projectName: 'proj', name: 'schedule-123' });
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.suspendScheduledSession).toHaveBeenCalledWith('proj', 'schedule-123');
    expect(result.current.data?.suspend).toBe(true);
    expect(queryClient.getQueryState(scheduledSessionKeys.detail('proj', 'schedule-123'))?.isInvalidated).toBe(true);
    expect(queryClient.getQueryState(scheduledSessionKeys.list('proj'))?.isInvalidated).toBe(true);
  });
});

describe('useResumeScheduledSession', () => {
  it('resumes a scheduled session and invalidates caches', async () => {
    const queryClient = createTestQueryClient();
    const fakePort = createFakeScheduledSessionsPort();

    queryClient.setQueryData(scheduledSessionKeys.detail('proj', 'schedule-123'), { ...mockSchedule, suspend: true });
    queryClient.setQueryData(scheduledSessionKeys.list('proj'), [{ ...mockSchedule, suspend: true }]);

    const { result } = renderHook(() => useResumeScheduledSession(fakePort), {
      wrapper: createWrapper(queryClient),
    });

    act(() => {
      result.current.mutate({ projectName: 'proj', name: 'schedule-123' });
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.resumeScheduledSession).toHaveBeenCalledWith('proj', 'schedule-123');
    expect(result.current.data?.suspend).toBe(false);
    expect(queryClient.getQueryState(scheduledSessionKeys.detail('proj', 'schedule-123'))?.isInvalidated).toBe(true);
    expect(queryClient.getQueryState(scheduledSessionKeys.list('proj'))?.isInvalidated).toBe(true);
  });
});

describe('useTriggerScheduledSession', () => {
  it('triggers a scheduled session and invalidates runs cache', async () => {
    const queryClient = createTestQueryClient();
    const fakePort = createFakeScheduledSessionsPort();

    queryClient.setQueryData(scheduledSessionKeys.runs('proj', 'schedule-123'), []);

    const { result } = renderHook(() => useTriggerScheduledSession(fakePort), {
      wrapper: createWrapper(queryClient),
    });

    act(() => {
      result.current.mutate({ projectName: 'proj', name: 'schedule-123' });
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.triggerScheduledSession).toHaveBeenCalledWith('proj', 'schedule-123');
    expect(result.current.data?.name).toBe('schedule-123-manual-abc');
    expect(queryClient.getQueryState(scheduledSessionKeys.runs('proj', 'schedule-123'))?.isInvalidated).toBe(true);
  });
});
