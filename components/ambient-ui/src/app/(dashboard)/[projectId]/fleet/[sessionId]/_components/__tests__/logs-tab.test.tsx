import { describe, it, expect, vi, afterEach, beforeEach } from 'vitest'
import { render, screen, within, fireEvent, cleanup } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { createElement } from 'react'
import type { DomainSession, DomainSessionMessage } from '@/domain/types'

function makeSession(overrides: Partial<DomainSession> = {}): DomainSession {
  return {
    id: 'sess-001',
    name: 'test-session',
    phase: 'Running',
    agentId: null,
    agentName: null,
    projectId: 'proj-001',
    model: null,
    startTime: null,
    completionTime: null,
    createdAt: '2026-01-15T10:00:00Z',
    updatedAt: '2026-01-15T10:00:00Z',
    annotations: {},
    ...overrides,
  }
}

function makeMessage(overrides: Partial<DomainSessionMessage> = {}): DomainSessionMessage {
  return {
    id: 'msg-001',
    sessionId: 'sess-001',
    eventType: 'tool_use',
    payload: 'ls -la /workspace',
    seq: 1,
    createdAt: new Date(Date.now() - 60_000).toISOString(),
    ...overrides,
  }
}

function toSdkResponse(messages: DomainSessionMessage[]) {
  const sdkMessages = messages.map((m) => ({
    id: m.id,
    kind: 'SessionMessage',
    href: `/api/ambient/v1/sessions/${m.sessionId}/messages/${m.id}`,
    created_at: m.createdAt,
    updated_at: m.createdAt,
    session_id: m.sessionId,
    event_type: m.eventType,
    payload: m.payload,
    seq: m.seq,
  }))
  return {
    kind: 'SessionMessageList',
    page: 1,
    size: 100,
    total: sdkMessages.length,
    items: sdkMessages,
  }
}

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  return function Wrapper({ children }: { children: React.ReactNode }) {
    return createElement(QueryClientProvider, { client: queryClient }, children)
  }
}

describe('LogsTab', () => {
  const originalFetch = globalThis.fetch

  beforeEach(() => {
    vi.resetModules()
  })

  afterEach(() => {
    cleanup()
    globalThis.fetch = originalFetch
  })

  /**
   * Replace globalThis.fetch BEFORE importing the component so that
   * the adapter singleton's `fetch.bind(globalThis)` captures our fake.
   */
  async function loadWithMessages(messages: DomainSessionMessage[]) {
    const response = toSdkResponse(messages)

    globalThis.fetch = (async () => ({
      ok: true,
      status: 200,
      json: async () => response,
    })) as unknown as typeof fetch

    const { LogsTab } = await import('../logs-tab')
    return LogsTab
  }

  it('renders filter buttons for operational event types', async () => {
    const LogsTab = await loadWithMessages([])

    render(<LogsTab session={makeSession()} />, { wrapper: createWrapper() })

    expect(screen.getByRole('button', { name: 'Tool Call' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Tool Result' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Error' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Lifecycle' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'System' })).toBeInTheDocument()
  })

  it('renders filter buttons within a group with proper aria-label', async () => {
    const LogsTab = await loadWithMessages([])

    render(<LogsTab session={makeSession()} />, { wrapper: createWrapper() })

    const filterGroup = screen.getByRole('group', { name: 'Filter by event type' })
    expect(filterGroup).toBeInTheDocument()

    const buttons = within(filterGroup).getAllByRole('button')
    expect(buttons.length).toBe(5)
  })

  it('shows "No events recorded yet." when there are no messages', async () => {
    const LogsTab = await loadWithMessages([])

    render(<LogsTab session={makeSession()} />, { wrapper: createWrapper() })

    const emptyText = await screen.findByText('No events recorded yet.')
    expect(emptyText).toBeInTheDocument()
  })

  it('renders messages that match active filters', async () => {
    const messages = [
      makeMessage({ id: 'msg-1', eventType: 'tool_use', payload: 'git status' }),
      makeMessage({ id: 'msg-2', eventType: 'error', payload: 'Command failed' }),
    ]
    const LogsTab = await loadWithMessages(messages)

    render(<LogsTab session={makeSession()} />, { wrapper: createWrapper() })

    expect(await screen.findByText('git status')).toBeInTheDocument()
    expect(screen.getByText('Command failed')).toBeInTheDocument()
  })

  it('hides messages when their filter is toggled off', async () => {
    const messages = [
      makeMessage({ id: 'msg-1', eventType: 'tool_use', payload: 'git status' }),
      makeMessage({ id: 'msg-2', eventType: 'error', payload: 'Command failed' }),
    ]
    const LogsTab = await loadWithMessages(messages)

    render(<LogsTab session={makeSession()} />, { wrapper: createWrapper() })

    // Wait for messages to load
    expect(await screen.findByText('git status')).toBeInTheDocument()

    // Toggle off 'Tool Call' filter -- scope to filter group to avoid
    // matching EventTypeBadge elements in message rows
    const filterGroup = screen.getByRole('group', { name: 'Filter by event type' })
    fireEvent.click(within(filterGroup).getByText('Tool Call'))

    // tool_use message should be hidden
    expect(screen.queryByText('git status')).not.toBeInTheDocument()
    // error message should still be visible
    expect(screen.getByText('Command failed')).toBeInTheDocument()
  })

  it('shows "No events match" when all filters toggled off', async () => {
    const messages = [
      makeMessage({ id: 'msg-1', eventType: 'tool_use', payload: 'some command' }),
    ]
    const LogsTab = await loadWithMessages(messages)

    render(<LogsTab session={makeSession()} />, { wrapper: createWrapper() })

    // Wait for data to load
    expect(await screen.findByText('some command')).toBeInTheDocument()

    // Toggle off all filters using the filter group buttons.
    // We must scope to the filter group because EventTypeBadge also
    // renders these labels inside message rows.
    const filterGroup = screen.getByRole('group', { name: 'Filter by event type' })
    const filterButtons = within(filterGroup).getAllByRole('button')
    for (const btn of filterButtons) {
      fireEvent.click(btn)
    }

    expect(
      screen.getByText('No events match the selected filters.'),
    ).toBeInTheDocument()
  })
})
