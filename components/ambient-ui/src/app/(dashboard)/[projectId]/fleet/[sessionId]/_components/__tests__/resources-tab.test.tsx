import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { ResourcesTab } from '../resources-tab'
import type { DomainSession, DomainRepo, DomainReconciledRepo } from '@/domain/types'

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

const REPO: DomainRepo = {
  url: 'https://github.com/org/platform.git',
  branch: 'main',
  name: 'platform',
  autoPush: false,
}

const RECONCILED: DomainReconciledRepo = {
  url: 'https://github.com/org/platform.git',
  name: 'platform',
  status: 'Ready',
  currentActiveBranch: 'feat/new-feature',
  defaultBranch: 'main',
  clonedAt: '2026-01-15T10:02:00Z',
}

describe('ResourcesTab', () => {
  it('shows empty state when no repos', () => {
    render(<ResourcesTab session={makeSession()} />)
    expect(screen.getByText('No resources attached')).toBeTruthy()
    expect(screen.getByText('This session has no repositories configured.')).toBeTruthy()
  })

  it('renders repo table with merged data', () => {
    render(
      <ResourcesTab
        session={makeSession({ repos: [REPO], reconciledRepos: [RECONCILED] })}
      />,
    )
    expect(screen.getByText('platform')).toBeTruthy()
    const link = screen.getByRole('link', { name: 'https://github.com/org/platform.git' })
    expect(link).toBeTruthy()
    expect(link.getAttribute('href')).toBe('https://github.com/org/platform.git')
    expect(link.getAttribute('target')).toBe('_blank')
    expect(screen.getByText('feat/new-feature')).toBeTruthy()
    expect(screen.getByText('Ready')).toBeTruthy()
  })

  it('shows config branch when no reconciled data', () => {
    render(<ResourcesTab session={makeSession({ repos: [REPO] })} />)
    expect(screen.getByText('main')).toBeTruthy()
  })

  it('renders clone status badges with correct text', () => {
    const cloningRepo: DomainReconciledRepo = { ...RECONCILED, status: 'Cloning', clonedAt: null }
    render(
      <ResourcesTab
        session={makeSession({ repos: [REPO], reconciledRepos: [cloningRepo] })}
      />,
    )
    expect(screen.getByText('Cloning')).toBeTruthy()
  })

  it('renders failed clone status', () => {
    const failedRepo: DomainReconciledRepo = { ...RECONCILED, status: 'Failed', clonedAt: null }
    render(
      <ResourcesTab
        session={makeSession({ repos: [REPO], reconciledRepos: [failedRepo] })}
      />,
    )
    expect(screen.getByText('Failed')).toBeTruthy()
  })

  it('shows dash for missing clone status', () => {
    const noStatusRepo: DomainReconciledRepo = { ...RECONCILED, status: null, clonedAt: null }
    render(
      <ResourcesTab
        session={makeSession({ repos: [REPO], reconciledRepos: [noStatusRepo] })}
      />,
    )
    const cells = screen.getAllByRole('cell')
    const statusCell = cells[3]
    expect(statusCell.textContent).toBe('—')
  })

  it('shows repository count in card title', () => {
    render(
      <ResourcesTab session={makeSession({ repos: [REPO] })} />,
    )
    expect(screen.getByText(/Repositories \(1\)/)).toBeTruthy()
  })

  it('renders repo URLs as clickable links with title tooltips', () => {
    render(
      <ResourcesTab session={makeSession({ repos: [REPO] })} />,
    )
    const link = screen.getByRole('link', { name: 'https://github.com/org/platform.git' })
    expect(link.getAttribute('title')).toBe('https://github.com/org/platform.git')
  })

  it('derives name from URL basename when no name provided', () => {
    const unnamedRepo: DomainRepo = { url: 'https://github.com/org/myrepo.git', branch: null, name: null, autoPush: false }
    render(<ResourcesTab session={makeSession({ repos: [unnamedRepo] })} />)
    expect(screen.getByText('myrepo')).toBeTruthy()
  })
})
