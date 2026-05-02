import { describe, it, expect, vi } from 'vitest'
import { createScheduledSessionsAdapter } from '../scheduled-sessions'
import type { ScheduledSession } from '@/types/api'

const fakeScheduled: ScheduledSession = {
  name: 'sched-1',
  namespace: 'ns',
  creationTimestamp: '2024-01-01T00:00:00Z',
  schedule: '0 9 * * 1-5',
  suspend: false,
  displayName: 'Daily check',
  sessionTemplate: {},
  activeCount: 0,
  reuseLastSession: false,
}

describe('scheduledSessionsAdapter', () => {
  it('delegates listScheduledSessions to API', async () => {
    const fakeApi = {
      listScheduledSessions: vi.fn().mockResolvedValue([fakeScheduled]),
      getScheduledSession: vi.fn(),
      createScheduledSession: vi.fn(),
      updateScheduledSession: vi.fn(),
      deleteScheduledSession: vi.fn(),
      suspendScheduledSession: vi.fn(),
      resumeScheduledSession: vi.fn(),
      triggerScheduledSession: vi.fn(),
      listScheduledSessionRuns: vi.fn(),
    }
    const adapter = createScheduledSessionsAdapter(fakeApi)

    const result = await adapter.listScheduledSessions('p')

    expect(result).toHaveLength(1)
    expect(result[0].name).toBe('sched-1')
  })

  it('delegates triggerScheduledSession to API', async () => {
    const fakeApi = {
      listScheduledSessions: vi.fn(),
      getScheduledSession: vi.fn(),
      createScheduledSession: vi.fn(),
      updateScheduledSession: vi.fn(),
      deleteScheduledSession: vi.fn(),
      suspendScheduledSession: vi.fn(),
      resumeScheduledSession: vi.fn(),
      triggerScheduledSession: vi.fn().mockResolvedValue({ name: 'run-1', namespace: 'ns' }),
      listScheduledSessionRuns: vi.fn(),
    }
    const adapter = createScheduledSessionsAdapter(fakeApi)

    const result = await adapter.triggerScheduledSession('p', 'sched-1')

    expect(result.name).toBe('run-1')
  })
})
