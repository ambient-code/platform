import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { FleetSummary } from '../fleet-summary'
import type { DomainSession, SessionPhase } from '@/domain/types'

function makeSession(overrides: Partial<DomainSession> = {}): DomainSession {
  return {
    id: 'sess-001',
    name: 'test-session',
    phase: 'Running',
    agentId: null,
    agentName: null,
    projectId: 'proj-001',
    model: null,
    temperature: null,
    maxTokens: null,
    timeout: null,
    workflowId: null,
    prompt: null,
    sdkRestartCount: 0,
    startTime: null,
    completionTime: null,
    createdAt: '2026-01-15T10:00:00Z',
    updatedAt: '2026-01-15T10:00:00Z',
    annotations: {},
    labels: {},
    environmentVariables: {},
    repos: [],
    reconciledRepos: [],
    conditions: [],
    ...overrides,
  }
}

describe('FleetSummary', () => {
  it('shows total session count', () => {
    const sessions = [
      makeSession({ id: 'sess-1' }),
      makeSession({ id: 'sess-2' }),
      makeSession({ id: 'sess-3' }),
    ]

    render(<FleetSummary sessions={sessions} />)
    expect(screen.getByText('3 sessions')).toBeInTheDocument()
  })

  it('shows phase counts grouped by phase', () => {
    const sessions = [
      makeSession({ id: 'sess-1', phase: 'Running' }),
      makeSession({ id: 'sess-2', phase: 'Running' }),
      makeSession({ id: 'sess-3', phase: 'Failed' }),
      makeSession({ id: 'sess-4', phase: 'Completed' }),
    ]

    render(<FleetSummary sessions={sessions} />)
    expect(screen.getByText('4 sessions')).toBeInTheDocument()
    expect(screen.getByText('Running')).toBeInTheDocument()
    expect(screen.getByText('Failed')).toBeInTheDocument()
    expect(screen.getByText('Completed')).toBeInTheDocument()
    expect(screen.getByText('2')).toBeInTheDocument()
    expect(screen.getAllByText('1')).toHaveLength(2)
  })

  it('does not render phases with zero count', () => {
    const sessions = [makeSession({ id: 'sess-1', phase: 'Running' })]

    render(<FleetSummary sessions={sessions} />)
    expect(screen.queryByText('Pending')).not.toBeInTheDocument()
    expect(screen.queryByText('Failed')).not.toBeInTheDocument()
    expect(screen.queryByText('Stopped')).not.toBeInTheDocument()
  })

  it('handles empty sessions array', () => {
    render(<FleetSummary sessions={[]} />)
    expect(screen.getByText('0 sessions')).toBeInTheDocument()
  })

  it('shows filtered count when filteredCount differs from total', () => {
    const sessions = [
      makeSession({ id: 'sess-1' }),
      makeSession({ id: 'sess-2' }),
      makeSession({ id: 'sess-3' }),
    ]

    render(<FleetSummary sessions={sessions} filteredCount={2} />)
    expect(screen.getByText('Showing 2 of 3 sessions')).toBeInTheDocument()
  })

  it('shows normal count when filteredCount equals total', () => {
    const sessions = [
      makeSession({ id: 'sess-1' }),
      makeSession({ id: 'sess-2' }),
    ]

    render(<FleetSummary sessions={sessions} filteredCount={2} />)
    expect(screen.getByText('2 sessions')).toBeInTheDocument()
  })

  it('calls onPhaseFilter when a phase chip is clicked', () => {
    const onPhaseFilter = vi.fn()
    const sessions = [
      makeSession({ id: 'sess-1', phase: 'Running' }),
      makeSession({ id: 'sess-2', phase: 'Failed' }),
    ]

    render(
      <FleetSummary
        sessions={sessions}
        onPhaseFilter={onPhaseFilter}
      />
    )

    const runningButton = screen.getByRole('button', { name: 'Filter by Running' })
    fireEvent.click(runningButton)
    expect(onPhaseFilter).toHaveBeenCalledWith('Running')
  })

  it('clears phase filter when active phase chip is clicked', () => {
    const onPhaseFilter = vi.fn()
    const sessions = [
      makeSession({ id: 'sess-1', phase: 'Running' }),
    ]

    render(
      <FleetSummary
        sessions={sessions}
        activePhase={'Running' as SessionPhase}
        onPhaseFilter={onPhaseFilter}
      />
    )

    const runningButton = screen.getByRole('button', { name: 'Filter by Running' })
    fireEvent.click(runningButton)
    expect(onPhaseFilter).toHaveBeenCalledWith(null)
  })

  it('renders phase chips as buttons when onPhaseFilter is provided', () => {
    const sessions = [
      makeSession({ id: 'sess-1', phase: 'Running' }),
    ]

    render(
      <FleetSummary
        sessions={sessions}
        onPhaseFilter={() => {}}
      />
    )

    expect(screen.getByRole('button', { name: 'Filter by Running' })).toBeInTheDocument()
  })

  it('renders phase chips as non-interactive when onPhaseFilter is not provided', () => {
    const sessions = [
      makeSession({ id: 'sess-1', phase: 'Running' }),
    ]

    render(<FleetSummary sessions={sessions} />)
    expect(screen.queryByRole('button', { name: 'Filter by Running' })).not.toBeInTheDocument()
  })
})
