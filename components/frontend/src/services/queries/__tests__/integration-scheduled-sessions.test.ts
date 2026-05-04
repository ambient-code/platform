import { renderHook, waitFor, act } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { createScheduledSessionsAdapter } from '../../adapters/scheduled-sessions';
import {
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
import { createWrapper } from './test-utils';

const recordedSchedule = {
  name: 'schedule-123',
  namespace: 'proj-ns',
  creationTimestamp: '2026-01-01T00:00:00Z',
  schedule: '0 9 * * *',
  suspend: false,
  displayName: 'Daily build',
  sessionTemplate: { initialPrompt: 'Run tests' },
  activeCount: 0,
};

function createFakeScheduledSessionsApi() {
  return {
    listScheduledSessions: vi.fn().mockResolvedValue([recordedSchedule]),
    getScheduledSession: vi.fn().mockResolvedValue({ ...recordedSchedule, activeCount: 1 }),
    createScheduledSession: vi.fn().mockResolvedValue({
      ...recordedSchedule,
      name: 'schedule-456',
      displayName: 'New schedule',
    }),
    updateScheduledSession: vi.fn().mockResolvedValue({
      ...recordedSchedule,
      schedule: '0 10 * * *',
    }),
    deleteScheduledSession: vi.fn().mockResolvedValue(undefined),
    suspendScheduledSession: vi.fn().mockResolvedValue({ ...recordedSchedule, suspend: true }),
    resumeScheduledSession: vi.fn().mockResolvedValue({ ...recordedSchedule, suspend: false }),
    triggerScheduledSession: vi.fn().mockResolvedValue({ name: 'schedule-123-manual-xyz', namespace: 'proj' }),
    listScheduledSessionRuns: vi.fn().mockResolvedValue([
      { metadata: { name: 'run-1' }, status: { phase: 'Running' } },
    ]),
  };
}

describe('integration: hook → scheduledSessionsAdapter → fakeApi', () => {
  it('useScheduledSessions: list flows through', async () => {
    const fakeApi = createFakeScheduledSessionsApi();
    const adapter = createScheduledSessionsAdapter(fakeApi);

    const { result } = renderHook(
      () => useScheduledSessions('proj', adapter),
      { wrapper: createWrapper() },
    );

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.listScheduledSessions).toHaveBeenCalledWith('proj');
    expect(result.current.data).toHaveLength(1);
    expect(result.current.data?.[0].displayName).toBe('Daily build');
  });

  it('useScheduledSession: single item flows through', async () => {
    const fakeApi = createFakeScheduledSessionsApi();
    const adapter = createScheduledSessionsAdapter(fakeApi);

    const { result } = renderHook(
      () => useScheduledSession('proj', 'schedule-123', adapter),
      { wrapper: createWrapper() },
    );

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.getScheduledSession).toHaveBeenCalledWith('proj', 'schedule-123');
    expect(result.current.data?.activeCount).toBe(1);
  });

  it('useScheduledSessionRuns: runs flow through', async () => {
    const fakeApi = createFakeScheduledSessionsApi();
    const adapter = createScheduledSessionsAdapter(fakeApi);

    const { result } = renderHook(
      () => useScheduledSessionRuns('proj', 'schedule-123', adapter),
      { wrapper: createWrapper() },
    );

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.listScheduledSessionRuns).toHaveBeenCalledWith('proj', 'schedule-123');
    expect(result.current.data?.[0].metadata.name).toBe('run-1');
  });

  it('useCreateScheduledSession: create flows through', async () => {
    const fakeApi = createFakeScheduledSessionsApi();
    const adapter = createScheduledSessionsAdapter(fakeApi);

    const { result } = renderHook(
      () => useCreateScheduledSession(adapter),
      { wrapper: createWrapper() },
    );

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
    expect(fakeApi.createScheduledSession).toHaveBeenCalledWith(
      'proj',
      expect.objectContaining({ schedule: '0 12 * * 1' }),
    );
    expect(result.current.data?.name).toBe('schedule-456');
    expect(result.current.data?.displayName).toBe('New schedule');
  });

  it('useUpdateScheduledSession: update flows through', async () => {
    const fakeApi = createFakeScheduledSessionsApi();
    const adapter = createScheduledSessionsAdapter(fakeApi);

    const { result } = renderHook(
      () => useUpdateScheduledSession(adapter),
      { wrapper: createWrapper() },
    );

    act(() => {
      result.current.mutate({
        projectName: 'proj',
        name: 'schedule-123',
        data: { schedule: '0 10 * * *' },
      });
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.updateScheduledSession).toHaveBeenCalledWith('proj', 'schedule-123', { schedule: '0 10 * * *' });
    expect(result.current.data?.schedule).toBe('0 10 * * *');
  });

  it('useDeleteScheduledSession: delete flows through', async () => {
    const fakeApi = createFakeScheduledSessionsApi();
    const adapter = createScheduledSessionsAdapter(fakeApi);

    const { result } = renderHook(
      () => useDeleteScheduledSession(adapter),
      { wrapper: createWrapper() },
    );

    act(() => {
      result.current.mutate({ projectName: 'proj', name: 'schedule-123' });
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.deleteScheduledSession).toHaveBeenCalledWith('proj', 'schedule-123');
  });

  it('useSuspendScheduledSession: suspend flows through', async () => {
    const fakeApi = createFakeScheduledSessionsApi();
    const adapter = createScheduledSessionsAdapter(fakeApi);

    const { result } = renderHook(
      () => useSuspendScheduledSession(adapter),
      { wrapper: createWrapper() },
    );

    act(() => {
      result.current.mutate({ projectName: 'proj', name: 'schedule-123' });
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.suspendScheduledSession).toHaveBeenCalledWith('proj', 'schedule-123');
    expect(result.current.data?.suspend).toBe(true);
  });

  it('useResumeScheduledSession: resume flows through', async () => {
    const fakeApi = createFakeScheduledSessionsApi();
    const adapter = createScheduledSessionsAdapter(fakeApi);

    const { result } = renderHook(
      () => useResumeScheduledSession(adapter),
      { wrapper: createWrapper() },
    );

    act(() => {
      result.current.mutate({ projectName: 'proj', name: 'schedule-123' });
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.resumeScheduledSession).toHaveBeenCalledWith('proj', 'schedule-123');
    expect(result.current.data?.suspend).toBe(false);
  });

  it('useTriggerScheduledSession: trigger flows through', async () => {
    const fakeApi = createFakeScheduledSessionsApi();
    const adapter = createScheduledSessionsAdapter(fakeApi);

    const { result } = renderHook(
      () => useTriggerScheduledSession(adapter),
      { wrapper: createWrapper() },
    );

    act(() => {
      result.current.mutate({ projectName: 'proj', name: 'schedule-123' });
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.triggerScheduledSession).toHaveBeenCalledWith('proj', 'schedule-123');
    expect(result.current.data?.name).toBe('schedule-123-manual-xyz');
  });
});
