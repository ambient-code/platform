import { describe, it, expect, vi } from 'vitest'
import { createSessionsAdapter } from '../sessions'
import type { AgenticSession, ListAgenticSessionsPaginatedResponse } from '@/types/api'

const fakeSession: AgenticSession = {
  metadata: { name: 'test-session', namespace: 'test-ns', uid: 'uid-1', creationTimestamp: '2024-01-01T00:00:00Z' },
  spec: { llmSettings: { model: 'claude-sonnet', temperature: 0.7, maxTokens: 4096 }, timeout: 3600 },
  status: { phase: 'Running', startTime: '2024-01-01T00:00:00Z' },
}

const fakePaginatedResponse: ListAgenticSessionsPaginatedResponse = {
  items: [fakeSession],
  totalCount: 1,
  limit: 20,
  offset: 0,
  hasMore: false,
}

function makeFakeApi() {
  return {
    listSessionsPaginated: vi.fn().mockResolvedValue(fakePaginatedResponse),
    listSessions: vi.fn().mockResolvedValue([fakeSession]),
    getSession: vi.fn().mockResolvedValue(fakeSession),
    createSession: vi.fn().mockResolvedValue(fakeSession),
    stopSession: vi.fn().mockResolvedValue('stopped'),
    startSession: vi.fn().mockResolvedValue({ message: 'started' }),
    cloneSession: vi.fn().mockResolvedValue(fakeSession),
    deleteSession: vi.fn().mockResolvedValue(undefined),
    getSessionPodEvents: vi.fn().mockResolvedValue({ events: [] }),
    updateSessionDisplayName: vi.fn().mockResolvedValue(fakeSession),
    updateSessionMcpServers: vi.fn().mockResolvedValue(fakeSession),
    getSessionExport: vi.fn().mockResolvedValue({ sessionId: 'test', projectName: 'p', exportDate: '', aguiEvents: [], hasLegacy: false }),
    switchSessionModel: vi.fn().mockResolvedValue(fakeSession),
    saveToGoogleDrive: vi.fn().mockResolvedValue({ content: 'ok' }),
    getMcpStatus: vi.fn(),
    getReposStatus: vi.fn(),
    getCapabilities: vi.fn(),
  }
}

describe('sessionsAdapter', () => {
  it('transforms paginated response to PaginatedResult', async () => {
    const api = makeFakeApi()
    const adapter = createSessionsAdapter(api)

    const result = await adapter.listSessions('project-1')

    expect(result.items).toHaveLength(1)
    expect(result.items[0].metadata.name).toBe('test-session')
    expect(result.totalCount).toBe(1)
    expect(result.hasMore).toBe(false)
    expect(result.nextPage).toBeUndefined()
  })

  it('provides nextPage when hasMore is true', async () => {
    const api = makeFakeApi()
    api.listSessionsPaginated
      .mockResolvedValueOnce({ ...fakePaginatedResponse, hasMore: true })
      .mockResolvedValueOnce(fakePaginatedResponse)
    const adapter = createSessionsAdapter(api)

    const result = await adapter.listSessions('project-1')
    expect(result.nextPage).toBeDefined()

    const page2 = await result.nextPage!()
    expect(page2.items).toHaveLength(1)
    expect(page2.nextPage).toBeUndefined()
  })

  it('delegates getSession to API', async () => {
    const api = makeFakeApi()
    const adapter = createSessionsAdapter(api)

    const result = await adapter.getSession('project-1', 'test-session')

    expect(result.metadata.name).toBe('test-session')
    expect(api.getSession).toHaveBeenCalledWith('project-1', 'test-session')
  })

  it('delegates createSession to API', async () => {
    const api = makeFakeApi()
    const adapter = createSessionsAdapter(api)
    const request = { initialPrompt: 'hello', llmSettings: { model: 'claude-sonnet' } }

    await adapter.createSession('project-1', request)

    expect(api.createSession).toHaveBeenCalledWith('project-1', request)
  })

  it('delegates stopSession to API', async () => {
    const api = makeFakeApi()
    const adapter = createSessionsAdapter(api)

    const result = await adapter.stopSession('project-1', 'test-session')

    expect(result).toBe('stopped')
  })

  it('delegates deleteSession to API', async () => {
    const api = makeFakeApi()
    const adapter = createSessionsAdapter(api)

    await adapter.deleteSession('project-1', 'test-session')

    expect(api.deleteSession).toHaveBeenCalledWith('project-1', 'test-session')
  })
})
