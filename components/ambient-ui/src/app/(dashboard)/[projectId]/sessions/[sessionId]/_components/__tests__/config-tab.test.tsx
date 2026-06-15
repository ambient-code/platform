import { describe, it, expect } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { ConfigTab } from '../config-tab'
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

describe('ConfigTab', () => {
  it('renders configuration metadata', () => {
    render(<ConfigTab session={makeSession()} />)
    expect(screen.getByText('Configuration')).toBeTruthy()
    expect(screen.getByText('claude-sonnet-4-20250514')).toBeTruthy()
    expect(screen.getByText('0.7')).toBeTruthy()
    expect(screen.getByText('4096')).toBeTruthy()
    expect(screen.getByText('3600s')).toBeTruthy()
  })

  it('shows dashes for null config values', () => {
    render(
      <ConfigTab
        session={makeSession({ model: null, temperature: null, maxTokens: null, timeout: null })}
      />,
    )
    const dashes = screen.getAllByText('—')
    expect(dashes.length).toBeGreaterThanOrEqual(4)
  })

  it('renders environment variables table with count', () => {
    render(
      <ConfigTab
        session={makeSession({ environmentVariables: { NODE_ENV: 'production', DEBUG: 'true' } })}
      />,
    )
    expect(screen.getByText('Environment Variables (2)')).toBeTruthy()
    expect(screen.getByText('NODE_ENV')).toBeTruthy()
    expect(screen.getByText('production')).toBeTruthy()
    expect(screen.getByText('DEBUG')).toBeTruthy()
  })

  it('hides environment variables section when empty', () => {
    render(<ConfigTab session={makeSession()} />)
    expect(screen.queryByText(/Environment Variables/)).toBeNull()
  })

  it('renders annotations with friendly labels for registered keys', () => {
    render(
      <ConfigTab
        session={makeSession({
          annotations: {
            'ambient-code.io/jira/issue': 'HYPERFLEET-234',
            'ambient-code.io/github/pr': 'org/repo#42',
          },
        })}
      />,
    )
    expect(screen.getByText('Annotations (2)')).toBeTruthy()
    expect(screen.getByText('Jira Issue')).toBeTruthy()
    expect(screen.getByText('GitHub PR')).toBeTruthy()
    const hyperfleetMatches = screen.getAllByText('HYPERFLEET-234')
    expect(hyperfleetMatches.length).toBeGreaterThanOrEqual(1)
    const prMatches = screen.getAllByText('org/repo#42')
    expect(prMatches.length).toBeGreaterThanOrEqual(1)
  })

  it('renders raw annotation keys when not registered', () => {
    render(
      <ConfigTab
        session={makeSession({
          annotations: { 'custom-key': 'custom-val' },
        })}
      />,
    )
    expect(screen.getByText('Annotations (1)')).toBeTruthy()
    expect(screen.getByText('custom-key')).toBeTruthy()
    expect(screen.getByText('custom-val')).toBeTruthy()
  })

  it('hides annotations section when no annotations exist', () => {
    render(<ConfigTab session={makeSession()} />)
    expect(screen.queryByText(/Annotations/)).toBeNull()
  })

  it('renders labels table with count', () => {
    render(
      <ConfigTab
        session={makeSession({ labels: { team: 'platform', tier: 'production' } })}
      />,
    )
    expect(screen.getByText('Labels (2)')).toBeTruthy()
    expect(screen.getByText('team')).toBeTruthy()
    expect(screen.getByText('platform')).toBeTruthy()
  })

  it('hides labels section when empty', () => {
    render(<ConfigTab session={makeSession()} />)
    expect(screen.queryByText(/Labels/)).toBeNull()
  })

  it('renders prompt with truncation and char count', () => {
    const longPrompt = 'x'.repeat(300)
    render(<ConfigTab session={makeSession({ prompt: longPrompt })} />)
    expect(screen.getByText('Prompt')).toBeTruthy()
    expect(screen.getByText('Show more (300 chars)')).toBeTruthy()
  })

  it('expands truncated prompt on click', () => {
    const longPrompt = 'A'.repeat(100) + 'B'.repeat(200)
    render(<ConfigTab session={makeSession({ prompt: longPrompt })} />)
    fireEvent.click(screen.getByText('Show more (300 chars)'))
    expect(screen.getByText('Show less')).toBeTruthy()
    expect(screen.getByText(longPrompt)).toBeTruthy()
  })

  it('renders short prompt without truncation', () => {
    render(<ConfigTab session={makeSession({ prompt: 'Fix the auth bug' })} />)
    expect(screen.getByText('Fix the auth bug')).toBeTruthy()
    expect(screen.queryByText(/Show more/)).toBeNull()
  })

  it('renders clickable URL annotation values as links', () => {
    render(
      <ConfigTab
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

  it('masks secret-looking env var values', () => {
    render(
      <ConfigTab
        session={makeSession({
          environmentVariables: { CREDENTIAL_ID: 'masked-val', NODE_ENV: 'production' },
        })}
      />,
    )
    expect(screen.getByText('NODE_ENV')).toBeTruthy()
    expect(screen.getByText('production')).toBeTruthy()
    expect(screen.getByText('CREDENTIAL_ID')).toBeTruthy()
    expect(screen.getByText('••••••••')).toBeTruthy()
    expect(screen.queryByText('masked-val')).toBeNull()
  })

  it('reveals secret value on toggle click', () => {
    render(
      <ConfigTab
        session={makeSession({
          environmentVariables: { SENSITIVE_CREDENTIAL: 'revealed-val' },
        })}
      />,
    )
    expect(screen.getByText('••••••••')).toBeTruthy()
    fireEvent.click(screen.getByLabelText('Reveal secret value'))
    expect(screen.getByText('revealed-val')).toBeTruthy()
    expect(screen.queryByText('••••••••')).toBeNull()
  })

  it('hides Agent Restarts when sdkRestartCount is 0', () => {
    render(<ConfigTab session={makeSession({ sdkRestartCount: 0 })} />)
    expect(screen.queryByText('Agent Restarts')).toBeNull()
  })

  it('shows Agent Restarts when sdkRestartCount > 0', () => {
    render(<ConfigTab session={makeSession({ sdkRestartCount: 3 })} />)
    expect(screen.getByText('Agent Restarts')).toBeTruthy()
    expect(screen.getByText('3')).toBeTruthy()
  })

  it('renders Workflow ID with mono styling and tooltip', () => {
    render(<ConfigTab session={makeSession({ workflowId: 'wf-abc-123' })} />)
    const wfElement = screen.getByText('wf-abc-123')
    expect(wfElement).toBeTruthy()
    expect(wfElement.getAttribute('title')).toBe('Workflow ID')
    expect(wfElement.className).toContain('font-mono')
  })
})
