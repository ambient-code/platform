import { describe, it, expect } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { DetailsTab } from '../details-tab'
import type { DomainSession } from '@/domain/types'

function makeSession(overrides: Partial<DomainSession> = {}): DomainSession {
  return {
    id: 'sess-001',
    name: 'test-session',
    phase: 'Running',
    agentId: null,
    agentName: null,
    projectId: 'proj-001',
    model: 'claude-sonnet-4-20250514',
    temperature: 0.7,
    maxTokens: 4096,
    timeout: 3600,
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

describe('DetailsTab', () => {
  it('renders configuration metadata', () => {
    render(<DetailsTab session={makeSession()} />)
    expect(screen.getByText('Configuration')).toBeTruthy()
    expect(screen.getByText('claude-sonnet-4-20250514')).toBeTruthy()
    expect(screen.getByText('0.7')).toBeTruthy()
    expect(screen.getByText('4096')).toBeTruthy()
    expect(screen.getByText('3600s')).toBeTruthy()
  })

  it('shows dashes for null config values', () => {
    render(
      <DetailsTab
        session={makeSession({ model: null, temperature: null, maxTokens: null, timeout: null })}
      />,
    )
    const dashes = screen.getAllByText('—')
    expect(dashes.length).toBeGreaterThanOrEqual(4)
  })

  it('renders environment variables table', () => {
    render(
      <DetailsTab
        session={makeSession({ environmentVariables: { NODE_ENV: 'production', DEBUG: 'true' } })}
      />,
    )
    expect(screen.getByText('Environment Variables')).toBeTruthy()
    expect(screen.getByText('NODE_ENV')).toBeTruthy()
    expect(screen.getByText('production')).toBeTruthy()
    expect(screen.getByText('DEBUG')).toBeTruthy()
  })

  it('hides environment variables section when empty', () => {
    render(<DetailsTab session={makeSession()} />)
    expect(screen.queryByText('Environment Variables')).toBeNull()
  })

  it('renders registered annotations with labels', () => {
    render(
      <DetailsTab
        session={makeSession({
          annotations: {
            'ambient-code.io/jira/issue': 'HYPERFLEET-234',
            'ambient-code.io/github/pr': 'org/repo#42',
          },
        })}
      />,
    )
    expect(screen.getByText('Registered Annotations')).toBeTruthy()
    expect(screen.getByText('Jira Issue')).toBeTruthy()
    expect(screen.getByText('GitHub PR')).toBeTruthy()
    const hyperfleetMatches = screen.getAllByText('HYPERFLEET-234')
    expect(hyperfleetMatches.length).toBeGreaterThanOrEqual(1)
    const prMatches = screen.getAllByText('org/repo#42')
    expect(prMatches.length).toBeGreaterThanOrEqual(1)
  })

  it('hides registered annotations when none match registry', () => {
    render(
      <DetailsTab
        session={makeSession({ annotations: { 'custom-key': 'value' } })}
      />,
    )
    expect(screen.queryByText('Registered Annotations')).toBeNull()
  })

  it('renders raw annotations table for all annotations', () => {
    render(
      <DetailsTab
        session={makeSession({
          annotations: { 'ambient-code.io/jira/issue': 'X-1', 'custom-key': 'custom-val' },
        })}
      />,
    )
    expect(screen.getByText('Raw Annotations')).toBeTruthy()
    expect(screen.getByText('custom-key')).toBeTruthy()
    expect(screen.getByText('custom-val')).toBeTruthy()
  })

  it('hides raw annotations when no annotations exist', () => {
    render(<DetailsTab session={makeSession()} />)
    expect(screen.queryByText('Raw Annotations')).toBeNull()
  })

  it('renders labels table', () => {
    render(
      <DetailsTab
        session={makeSession({ labels: { team: 'platform', tier: 'production' } })}
      />,
    )
    expect(screen.getByText('Labels')).toBeTruthy()
    expect(screen.getByText('team')).toBeTruthy()
    expect(screen.getByText('platform')).toBeTruthy()
  })

  it('hides labels section when empty', () => {
    render(<DetailsTab session={makeSession()} />)
    expect(screen.queryByText('Labels')).toBeNull()
  })

  it('renders prompt with truncation', () => {
    const longPrompt = 'x'.repeat(300)
    render(<DetailsTab session={makeSession({ prompt: longPrompt })} />)
    expect(screen.getByText('Prompt')).toBeTruthy()
    expect(screen.getByText('Show more')).toBeTruthy()
  })

  it('expands truncated prompt on click', () => {
    const longPrompt = 'A'.repeat(100) + 'B'.repeat(200)
    render(<DetailsTab session={makeSession({ prompt: longPrompt })} />)
    fireEvent.click(screen.getByText('Show more'))
    expect(screen.getByText('Show less')).toBeTruthy()
    expect(screen.getByText(longPrompt)).toBeTruthy()
  })

  it('renders short prompt without truncation', () => {
    render(<DetailsTab session={makeSession({ prompt: 'Fix the auth bug' })} />)
    expect(screen.getByText('Fix the auth bug')).toBeTruthy()
    expect(screen.queryByText('Show more')).toBeNull()
  })

  it('renders clickable URL annotation values as links', () => {
    render(
      <DetailsTab
        session={makeSession({
          annotations: { 'ambient-code.io/ui/preview-url': 'https://app.example.com' },
        })}
      />,
    )
    const link = screen.getByRole('link', { name: 'https://app.example.com' })
    expect(link).toBeTruthy()
    expect(link.getAttribute('href')).toBe('https://app.example.com')
    expect(link.getAttribute('target')).toBe('_blank')
  })
})
