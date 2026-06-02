import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
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
    startTime: null,
    completionTime: null,
    createdAt: '2026-01-15T10:00:00Z',
    updatedAt: '2026-01-15T10:00:00Z',
    annotations: {},
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
})
