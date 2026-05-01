/* eslint-disable react/display-name */
import { renderHook, waitFor, act } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { describe, it, expect, vi } from 'vitest';
import React from 'react';
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

vi.mock('@/services/api/scheduled-sessions', () => ({
  listScheduledSessions: vi.fn().mockResolvedValue([
    {
      name: 'schedule-123',
      namespace: 'proj-ns',
      creationTimestamp: '2026-01-01T00:00:00Z',
      schedule: '0 9 * * *',
      suspend: false,
      displayName: 'Daily build',
      sessionTemplate: { initialPrompt: 'Run tests' },
      activeCount: 0,
    },
  ]),
  createScheduledSession: vi.fn().mockResolvedValue({
    name: 'schedule-456',
    namespace: 'proj-ns',
    creationTimestamp: '2026-01-02T00:00:00Z',
    schedule: '0 12 * * 1',
    suspend: false,
    displayName: 'Weekly review',
    sessionTemplate: { initialPrompt: 'Review code' },
    activeCount: 0,
  }),
  getScheduledSession: vi.fn().mockResolvedValue({
    name: 'schedule-123',
    namespace: 'proj-ns',
    creationTimestamp: '2026-01-01T00:00:00Z',
    schedule: '0 9 * * *',
    suspend: false,
    displayName: 'Daily build',
    sessionTemplate: { initialPrompt: 'Run tests' },
    activeCount: 1,
  }),
  updateScheduledSession: vi.fn().mockResolvedValue({
    name: 'schedule-123',
    namespace: 'proj-ns',
    creationTimestamp: '2026-01-01T00:00:00Z',
    schedule: '0 10 * * *',
    suspend: false,
    displayName: 'Updated schedule',
    sessionTemplate: { initialPrompt: 'Run tests' },
    activeCount: 0,
  }),
  deleteScheduledSession: vi.fn().mockResolvedValue(undefined),
  suspendScheduledSession: vi.fn().mockResolvedValue({
    name: 'schedule-123',
    namespace: 'proj-ns',
    creationTimestamp: '2026-01-01T00:00:00Z',
    schedule: '0 9 * * *',
    suspend: true,
    displayName: 'Daily build',
    sessionTemplate: { initialPrompt: 'Run tests' },
    activeCount: 0,
  }),
  resumeScheduledSession: vi.fn().mockResolvedValue({
    name: 'schedule-123',
    namespace: 'proj-ns',
    creationTimestamp: '2026-01-01T00:00:00Z',
    schedule: '0 9 * * *',
    suspend: false,
    displayName: 'Daily build',
    sessionTemplate: { initialPrompt: 'Run tests' },
    activeCount: 0,
  }),
  triggerScheduledSession: vi.fn().mockResolvedValue({ name: 'schedule-123-manual-abc', namespace: 'proj' }),
  listScheduledSessionRuns: vi.fn().mockResolvedValue([
    { metadata: { name: 'run-1' }, status: { phase: 'Running' } },
  ]),
}));

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return ({ children }: { children: React.ReactNode }) =>
    React.createElement(QueryClientProvider, { client: queryClient }, children);
}

describe('scheduledSessionKeys', () => {
  it('generates correct query keys', () => {
    expect(scheduledSessionKeys.all).toEqual(['scheduled-sessions']);
    expect(scheduledSessionKeys.lists()).toEqual(['scheduled-sessions', 'list']);
    expect(scheduledSessionKeys.list('proj')).toEqual(['scheduled-sessions', 'list', 'proj']);
    expect(scheduledSessionKeys.details()).toEqual(['scheduled-sessions', 'detail']);
    expect(scheduledSessionKeys.detail('proj', 'sched-1')).toEqual([
      'scheduled-sessions',
      'detail',
      'proj',
      'sched-1',
    ]);
    expect(scheduledSessionKeys.runs('proj', 'sched-1')).toEqual([
      'scheduled-sessions',
      'detail',
      'proj',
      'sched-1',
      'runs',
    ]);
  });
});

describe('useScheduledSessions', () => {
  it('fetches scheduled sessions list', async () => {
    const wrapper = createWrapper();
    const { result } = renderHook(() => useScheduledSessions('proj'), { wrapper });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toHaveLength(1);
    expect(result.current.data?.[0].name).toBe('schedule-123');
  });

  it('is disabled when projectName is empty', () => {
    const wrapper = createWrapper();
    const { result } = renderHook(() => useScheduledSessions(''), { wrapper });
    expect(result.current.fetchStatus).toBe('idle');
  });
});

describe('useScheduledSession', () => {
  it('fetches a single scheduled session', async () => {
    const wrapper = createWrapper();
    const { result } = renderHook(() => useScheduledSession('proj', 'schedule-123'), { wrapper });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data?.name).toBe('schedule-123');
  });

  it('is disabled when projectName or name is empty', () => {
    const wrapper = createWrapper();
    const { result } = renderHook(() => useScheduledSession('', 'schedule-123'), { wrapper });
    expect(result.current.fetchStatus).toBe('idle');

    const { result: result2 } = renderHook(() => useScheduledSession('proj', ''), { wrapper });
    expect(result2.current.fetchStatus).toBe('idle');
  });
});

describe('useScheduledSessionRuns', () => {
  it('fetches scheduled session runs', async () => {
    const wrapper = createWrapper();
    const { result } = renderHook(() => useScheduledSessionRuns('proj', 'schedule-123'), {
      wrapper,
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toHaveLength(1);
    expect(result.current.data?.[0].metadata.name).toBe('run-1');
  });

  it('is disabled when projectName or name is empty', () => {
    const wrapper = createWrapper();
    const { result } = renderHook(() => useScheduledSessionRuns('', 'schedule-123'), { wrapper });
    expect(result.current.fetchStatus).toBe('idle');

    const { result: result2 } = renderHook(() => useScheduledSessionRuns('proj', ''), { wrapper });
    expect(result2.current.fetchStatus).toBe('idle');
  });
});

describe('useCreateScheduledSession', () => {
  it('creates a scheduled session', async () => {
    const wrapper = createWrapper();
    const { result } = renderHook(() => useCreateScheduledSession(), { wrapper });

    act(() => {
      result.current.mutate({
        projectName: 'proj',
        data: {
          schedule: '0 12 * * 1',
          sessionTemplate: { initialPrompt: 'Review code' },
          // eslint-disable-next-line @typescript-eslint/no-explicit-any
        } as any,
      });
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data?.name).toBe('schedule-456');
  });
});

describe('useUpdateScheduledSession', () => {
  it('updates a scheduled session', async () => {
    const wrapper = createWrapper();
    const { result } = renderHook(() => useUpdateScheduledSession(), { wrapper });

    act(() => {
      result.current.mutate({
        projectName: 'proj',
        name: 'schedule-123',
        data: { schedule: '0 10 * * *' },
      });
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data?.schedule).toBe('0 10 * * *');
  });
});

describe('useDeleteScheduledSession', () => {
  it('deletes a scheduled session', async () => {
    const wrapper = createWrapper();
    const { result } = renderHook(() => useDeleteScheduledSession(), { wrapper });

    act(() => {
      result.current.mutate({
        projectName: 'proj',
        name: 'schedule-123',
      });
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
  });
});

describe('useSuspendScheduledSession', () => {
  it('suspends a scheduled session', async () => {
    const wrapper = createWrapper();
    const { result } = renderHook(() => useSuspendScheduledSession(), { wrapper });

    act(() => {
      result.current.mutate({
        projectName: 'proj',
        name: 'schedule-123',
      });
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data?.suspend).toBe(true);
  });
});

describe('useResumeScheduledSession', () => {
  it('resumes a scheduled session', async () => {
    const wrapper = createWrapper();
    const { result } = renderHook(() => useResumeScheduledSession(), { wrapper });

    act(() => {
      result.current.mutate({
        projectName: 'proj',
        name: 'schedule-123',
      });
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data?.suspend).toBe(false);
  });
});

describe('useTriggerScheduledSession', () => {
  it('triggers a scheduled session', async () => {
    const wrapper = createWrapper();
    const { result } = renderHook(() => useTriggerScheduledSession(), { wrapper });

    act(() => {
      result.current.mutate({
        projectName: 'proj',
        name: 'schedule-123',
      });
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data?.name).toBe('schedule-123-manual-abc');
  });
});
