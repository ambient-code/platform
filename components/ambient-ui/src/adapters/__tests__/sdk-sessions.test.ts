import { describe, it, expect } from 'vitest'
import type { Session, SessionList, ListOptions } from 'ambient-sdk'
import type { SessionsPort } from '@/ports/sessions'
import { createSessionsAdapter } from '../sdk-sessions'

function makeSdkSession(overrides: Partial<Session> = {}): Session {
  return {
    id: 'sess-001',
    kind: 'Session',
    href: '/api/ambient/v1/sessions/sess-001',
    created_at: '2026-01-15T10:00:00Z',
    updated_at: '2026-01-15T10:05:00Z',
    agent_id: 'agent-abc',
    annotations: '{"agent_name":"test-agent"}',
    assigned_user_id: '',
    bot_account_name: '',
    completion_time: '',
    conditions: '',
    created_by_user_id: '',
    environment_variables: '',
    kube_cr_name: '',
    kube_cr_uid: '',
    kube_namespace: '',
    labels: '',
    llm_max_tokens: 4096,
    llm_model: 'claude-sonnet-4-20250514',
    llm_temperature: 0.7,
    name: 'test-session',
    parent_session_id: '',
    phase: 'Running',
    project_id: 'proj-123',
    prompt: '',
    reconciled_repos: '',
    reconciled_workflow: '',
    repo_url: '',
    repos: '',
    resource_overrides: '',
    sdk_restart_count: 0,
    sdk_session_id: '',
    start_time: '2026-01-15T10:01:00Z',
    timeout: 3600,
    workflow_id: '',
    ...overrides,
  }
}

// Fake SessionAPI satisfying the shape the adapter needs
function createFakeSessionAPI(options: {
  sessions?: Session[]
  total?: number
  getResult?: Session
  startResult?: Session
  stopCalled?: { count: number }
}) {
  const sessions = options.sessions ?? [makeSdkSession()]
  const total = options.total ?? sessions.length

  return {
    list: async (): Promise<SessionList> => ({
      kind: 'SessionList',
      page: 1,
      size: 20,
      total,
      items: sessions,
    }),
    get: async (): Promise<Session> => {
      return options.getResult ?? sessions[0]
    },
    stop: async (): Promise<Session> => {
      if (options.stopCalled) options.stopCalled.count++
      return options.getResult ?? sessions[0]
    },
    start: async (): Promise<Session> => {
      return options.startResult ?? makeSdkSession({ phase: 'Pending' })
    },
    // Unused methods — included to satisfy the type shape
    create: async () => makeSdkSession(),
    update: async () => makeSdkSession(),
    delete: async () => undefined,
    updateStatus: async () => makeSdkSession(),
    listAll: async function* () { yield makeSdkSession() },
  }
}

describe('sdk-sessions adapter', () => {
  it('list() returns paginated domain sessions', async () => {
    const sessions = [
      makeSdkSession({ id: 'sess-001', name: 'first' }),
      makeSdkSession({ id: 'sess-002', name: 'second' }),
    ]
    const fakeAPI = createFakeSessionAPI({ sessions, total: 50 })
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const adapter: SessionsPort = createSessionsAdapter(fakeAPI as any)

    const result = await adapter.list('proj-123')

    expect(result.items).toHaveLength(2)
    expect(result.items[0].id).toBe('sess-001')
    expect(result.items[0].name).toBe('first')
    expect(result.items[0].phase).toBe('Running')
    expect(result.items[1].id).toBe('sess-002')
    expect(result.total).toBe(50)
    expect(result.page).toBe(1)
    expect(result.size).toBe(20)
    expect(result.hasMore).toBe(true)
  })

  it('list() returns hasMore=false when all items fit', async () => {
    const sessions = [makeSdkSession()]
    const fakeAPI = createFakeSessionAPI({ sessions, total: 1 })
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const adapter: SessionsPort = createSessionsAdapter(fakeAPI as any)

    const result = await adapter.list('proj-123')

    expect(result.hasMore).toBe(false)
  })

  it('list() maps SDK sessions to domain sessions', async () => {
    const sessions = [makeSdkSession({
      annotations: '{"agent_name":"my-agent"}',
      phase: 'Completed',
      completion_time: '2026-01-15T11:00:00Z',
    })]
    const fakeAPI = createFakeSessionAPI({ sessions })
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const adapter: SessionsPort = createSessionsAdapter(fakeAPI as any)

    const result = await adapter.list('proj-123')
    const session = result.items[0]

    expect(session.phase).toBe('Completed')
    expect(session.agentName).toBe('my-agent')
    expect(session.completionTime).toBe('2026-01-15T11:00:00Z')
  })

  it('get() returns a mapped domain session', async () => {
    const getResult = makeSdkSession({ id: 'sess-xyz', name: 'specific' })
    const fakeAPI = createFakeSessionAPI({ getResult })
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const adapter: SessionsPort = createSessionsAdapter(fakeAPI as any)

    const session = await adapter.get('sess-xyz')

    expect(session.id).toBe('sess-xyz')
    expect(session.name).toBe('specific')
  })

  it('stop() calls the API stop method', async () => {
    const stopCalled = { count: 0 }
    const fakeAPI = createFakeSessionAPI({ stopCalled })
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const adapter: SessionsPort = createSessionsAdapter(fakeAPI as any)

    await adapter.stop('sess-001')

    expect(stopCalled.count).toBe(1)
  })

  it('start() returns a mapped domain session', async () => {
    const startResult = makeSdkSession({ id: 'sess-start', phase: 'Pending' })
    const fakeAPI = createFakeSessionAPI({ startResult })
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const adapter: SessionsPort = createSessionsAdapter(fakeAPI as any)

    const session = await adapter.start('sess-start')

    expect(session.id).toBe('sess-start')
    expect(session.phase).toBe('Pending')
  })

  it('list() passes custom pagination params', async () => {
    let capturedOpts: ListOptions | undefined
    const fakeAPI = {
      ...createFakeSessionAPI({}),
      list: async (listOpts?: ListOptions): Promise<SessionList> => {
        capturedOpts = listOpts
        return { kind: 'SessionList', page: 2, size: 10, total: 25, items: [] }
      },
    }
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const adapter: SessionsPort = createSessionsAdapter(fakeAPI as any)

    await adapter.list('proj-123', { page: 2, size: 10 })

    expect(capturedOpts?.page).toBe(2)
    expect(capturedOpts?.size).toBe(10)
  })
})
