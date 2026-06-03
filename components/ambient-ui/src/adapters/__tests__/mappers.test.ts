import { describe, it, expect } from 'vitest'
import { mapSdkSessionToDomain, mapSdkProjectToDomain, mapSessionMessageToDomain } from '../mappers'
import type { SdkSessionMessageShape } from '../mappers'
import type { Session, Project } from 'ambient-sdk'

function makeSdkSession(overrides: Partial<Session> = {}): Session {
  return {
    id: 'sess-001',
    kind: 'Session',
    href: '/api/ambient/v1/sessions/sess-001',
    created_at: '2026-01-15T10:00:00Z',
    updated_at: '2026-01-15T10:05:00Z',
    agent_id: 'agent-abc',
    annotations: '{}',
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

function makeSdkProject(overrides: Partial<Project> = {}): Project {
  return {
    id: 'proj-001',
    kind: 'Project',
    href: '/api/ambient/v1/projects/proj-001',
    created_at: '2026-01-10T08:00:00Z',
    updated_at: '2026-01-10T09:00:00Z',
    annotations: '',
    description: 'A test project',
    labels: '',
    name: 'test-project',
    prompt: '',
    status: 'active',
    ...overrides,
  }
}

describe('mapSdkSessionToDomain', () => {
  it('maps snake_case fields to camelCase', () => {
    const sdk = makeSdkSession()
    const domain = mapSdkSessionToDomain(sdk)

    expect(domain.id).toBe('sess-001')
    expect(domain.name).toBe('test-session')
    expect(domain.agentId).toBe('agent-abc')
    expect(domain.projectId).toBe('proj-123')
    expect(domain.model).toBe('claude-sonnet-4-20250514')
    expect(domain.startTime).toBe('2026-01-15T10:01:00Z')
    expect(domain.completionTime).toBeNull()
    expect(domain.createdAt).toBe('2026-01-15T10:00:00Z')
    expect(domain.updatedAt).toBe('2026-01-15T10:05:00Z')
  })

  it('maps valid phase correctly', () => {
    for (const phase of ['Pending', 'Creating', 'Running', 'Stopping', 'Completed', 'Failed', 'Stopped'] as const) {
      const sdk = makeSdkSession({ phase })
      const domain = mapSdkSessionToDomain(sdk)
      expect(domain.phase).toBe(phase)
    }
  })

  it('defaults unknown phase to Pending', () => {
    const sdk = makeSdkSession({ phase: 'SomeUnknownPhase' })
    const domain = mapSdkSessionToDomain(sdk)
    expect(domain.phase).toBe('Pending')
  })

  it('defaults empty phase to Pending', () => {
    const sdk = makeSdkSession({ phase: '' })
    const domain = mapSdkSessionToDomain(sdk)
    expect(domain.phase).toBe('Pending')
  })

  it('parses valid annotations JSON to Record', () => {
    const annotations = JSON.stringify({
      agent_name: 'my-agent',
      team: 'platform',
    })
    const sdk = makeSdkSession({ annotations })
    const domain = mapSdkSessionToDomain(sdk)

    expect(domain.annotations).toEqual({
      agent_name: 'my-agent',
      team: 'platform',
    })
    expect(domain.agentName).toBe('my-agent')
  })

  it('returns empty Record for empty annotations string', () => {
    const sdk = makeSdkSession({ annotations: '' })
    const domain = mapSdkSessionToDomain(sdk)
    expect(domain.annotations).toEqual({})
    expect(domain.agentName).toBeNull()
  })

  it('returns empty Record for invalid JSON annotations', () => {
    const sdk = makeSdkSession({ annotations: 'not valid json {{{' })
    const domain = mapSdkSessionToDomain(sdk)
    expect(domain.annotations).toEqual({})
  })

  it('returns empty Record for JSON array annotations', () => {
    const sdk = makeSdkSession({ annotations: '["a", "b"]' })
    const domain = mapSdkSessionToDomain(sdk)
    expect(domain.annotations).toEqual({})
  })

  it('converts non-string annotation values to strings', () => {
    const annotations = JSON.stringify({
      count: 42,
      enabled: true,
    })
    const sdk = makeSdkSession({ annotations })
    const domain = mapSdkSessionToDomain(sdk)
    expect(domain.annotations).toEqual({
      count: '42',
      enabled: 'true',
    })
  })

  it('maps empty string fields to null', () => {
    const sdk = makeSdkSession({
      agent_id: '',
      project_id: '',
      llm_model: '',
      start_time: '',
      completion_time: '',
    })
    const domain = mapSdkSessionToDomain(sdk)

    expect(domain.agentId).toBeNull()
    expect(domain.projectId).toBeNull()
    expect(domain.model).toBeNull()
    expect(domain.startTime).toBeNull()
    expect(domain.completionTime).toBeNull()
  })

  it('handles null created_at and updated_at', () => {
    const sdk = makeSdkSession({
      created_at: null,
      updated_at: null,
    })
    const domain = mapSdkSessionToDomain(sdk)
    expect(domain.createdAt).toBe('')
    expect(domain.updatedAt).toBe('')
  })

  describe('new session fields', () => {
    it('parses repos from valid JSON string', () => {
      const repos = JSON.stringify([
        { url: 'https://github.com/org/repo1', branch: 'main', name: 'repo1', autoPush: true },
        { url: 'https://github.com/org/repo2', branch: null, name: null, autoPush: false },
      ])
      const sdk = makeSdkSession({ repos })
      const domain = mapSdkSessionToDomain(sdk)

      expect(domain.repos).toHaveLength(2)
      expect(domain.repos[0]).toEqual({
        url: 'https://github.com/org/repo1',
        branch: 'main',
        name: 'repo1',
        autoPush: true,
      })
      expect(domain.repos[1]).toEqual({
        url: 'https://github.com/org/repo2',
        branch: null,
        name: null,
        autoPush: false,
      })
    })

    it('returns empty repos for empty string', () => {
      const sdk = makeSdkSession({ repos: '' })
      const domain = mapSdkSessionToDomain(sdk)
      expect(domain.repos).toEqual([])
    })

    it('returns empty repos for invalid JSON', () => {
      const sdk = makeSdkSession({ repos: 'not valid json' })
      const domain = mapSdkSessionToDomain(sdk)
      expect(domain.repos).toEqual([])
    })

    it('parses reconciled repos with all status variants', () => {
      const reconciledRepos = JSON.stringify([
        { url: 'https://github.com/org/repo1', name: 'repo1', status: 'Cloning', currentActiveBranch: 'feat-1', defaultBranch: 'main', clonedAt: '2026-01-15T10:00:00Z' },
        { url: 'https://github.com/org/repo2', name: 'repo2', status: 'Ready', currentActiveBranch: 'main', defaultBranch: 'main', clonedAt: '2026-01-15T10:01:00Z' },
        { url: 'https://github.com/org/repo3', name: 'repo3', status: 'Failed', currentActiveBranch: null, defaultBranch: null, clonedAt: null },
      ])
      const sdk = makeSdkSession({ reconciled_repos: reconciledRepos })
      const domain = mapSdkSessionToDomain(sdk)

      expect(domain.reconciledRepos).toHaveLength(3)
      expect(domain.reconciledRepos[0]).toEqual({
        url: 'https://github.com/org/repo1',
        name: 'repo1',
        status: 'Cloning',
        currentActiveBranch: 'feat-1',
        defaultBranch: 'main',
        clonedAt: '2026-01-15T10:00:00Z',
      })
      expect(domain.reconciledRepos[1]!.status).toBe('Ready')
      expect(domain.reconciledRepos[2]!.status).toBe('Failed')
      expect(domain.reconciledRepos[2]!.currentActiveBranch).toBeNull()
      expect(domain.reconciledRepos[2]!.clonedAt).toBeNull()
    })

    it('returns null status for invalid reconciled repo status', () => {
      const reconciledRepos = JSON.stringify([
        { url: 'https://github.com/org/repo1', name: 'repo1', status: 'SomeBogusStatus' },
      ])
      const sdk = makeSdkSession({ reconciled_repos: reconciledRepos })
      const domain = mapSdkSessionToDomain(sdk)

      expect(domain.reconciledRepos).toHaveLength(1)
      expect(domain.reconciledRepos[0]!.status).toBeNull()
    })

    it('parses conditions array', () => {
      const conditions = JSON.stringify([
        { type: 'Ready', status: 'True', reason: 'AllGood', message: 'Session is ready', lastTransitionTime: '2026-01-15T10:05:00Z' },
        { type: 'Progressing', status: 'False', reason: null, message: null, lastTransitionTime: null },
      ])
      const sdk = makeSdkSession({ conditions })
      const domain = mapSdkSessionToDomain(sdk)

      expect(domain.conditions).toHaveLength(2)
      expect(domain.conditions[0]).toEqual({
        type: 'Ready',
        status: 'True',
        reason: 'AllGood',
        message: 'Session is ready',
        lastTransitionTime: '2026-01-15T10:05:00Z',
      })
      expect(domain.conditions[1]).toEqual({
        type: 'Progressing',
        status: 'False',
        reason: null,
        message: null,
        lastTransitionTime: null,
      })
    })

    it('returns Unknown for invalid condition status', () => {
      const conditions = JSON.stringify([
        { type: 'Ready', status: 'Maybe' },
      ])
      const sdk = makeSdkSession({ conditions })
      const domain = mapSdkSessionToDomain(sdk)

      expect(domain.conditions).toHaveLength(1)
      expect(domain.conditions[0]!.status).toBe('Unknown')
    })

    it('parses environment variables from JSON string', () => {
      const envVars = JSON.stringify({ NODE_ENV: 'production', API_URL: 'https://api.example.com' })
      const sdk = makeSdkSession({ environment_variables: envVars })
      const domain = mapSdkSessionToDomain(sdk)

      expect(domain.environmentVariables).toEqual({
        NODE_ENV: 'production',
        API_URL: 'https://api.example.com',
      })
    })

    it('parses labels from JSON string', () => {
      const labels = JSON.stringify({ team: 'platform', tier: 'production' })
      const sdk = makeSdkSession({ labels })
      const domain = mapSdkSessionToDomain(sdk)

      expect(domain.labels).toEqual({
        team: 'platform',
        tier: 'production',
      })
    })

    it('returns empty object for invalid env vars JSON', () => {
      const sdk = makeSdkSession({ environment_variables: '{broken' })
      const domain = mapSdkSessionToDomain(sdk)
      expect(domain.environmentVariables).toEqual({})
    })

    it('returns empty object for invalid labels JSON', () => {
      const sdk = makeSdkSession({ labels: 'not-json' })
      const domain = mapSdkSessionToDomain(sdk)
      expect(domain.labels).toEqual({})
    })

    it('returns empty object for array-shaped env vars', () => {
      const sdk = makeSdkSession({ environment_variables: '["a","b"]' })
      const domain = mapSdkSessionToDomain(sdk)
      expect(domain.environmentVariables).toEqual({})
    })

    it('returns empty object for array-shaped labels', () => {
      const sdk = makeSdkSession({ labels: '["x"]' })
      const domain = mapSdkSessionToDomain(sdk)
      expect(domain.labels).toEqual({})
    })

    it('maps temperature, maxTokens, timeout from SDK numbers', () => {
      const sdk = makeSdkSession({ llm_temperature: 0.5, llm_max_tokens: 8192, timeout: 7200 })
      const domain = mapSdkSessionToDomain(sdk)

      expect(domain.temperature).toBe(0.5)
      expect(domain.maxTokens).toBe(8192)
      expect(domain.timeout).toBe(7200)
    })

    it('returns null for zero temperature, maxTokens, timeout', () => {
      const sdk = makeSdkSession({ llm_temperature: 0, llm_max_tokens: 0, timeout: 0 })
      const domain = mapSdkSessionToDomain(sdk)

      expect(domain.temperature).toBeNull()
      expect(domain.maxTokens).toBeNull()
      expect(domain.timeout).toBeNull()
    })

    it('maps workflowId from workflow_id', () => {
      const sdk = makeSdkSession({ workflow_id: 'wf-42' })
      const domain = mapSdkSessionToDomain(sdk)
      expect(domain.workflowId).toBe('wf-42')
    })

    it('maps workflowId to null for empty string', () => {
      const sdk = makeSdkSession({ workflow_id: '' })
      const domain = mapSdkSessionToDomain(sdk)
      expect(domain.workflowId).toBeNull()
    })

    it('maps prompt from SDK', () => {
      const sdk = makeSdkSession({ prompt: 'Fix the bug in auth.ts' })
      const domain = mapSdkSessionToDomain(sdk)
      expect(domain.prompt).toBe('Fix the bug in auth.ts')
    })

    it('maps prompt to null for empty string', () => {
      const sdk = makeSdkSession({ prompt: '' })
      const domain = mapSdkSessionToDomain(sdk)
      expect(domain.prompt).toBeNull()
    })

    it('maps sdkRestartCount from sdk_restart_count', () => {
      const sdk = makeSdkSession({ sdk_restart_count: 3 })
      const domain = mapSdkSessionToDomain(sdk)
      expect(domain.sdkRestartCount).toBe(3)
    })

    it('defaults sdkRestartCount to 0 when sdk_restart_count is 0', () => {
      const sdk = makeSdkSession({ sdk_restart_count: 0 })
      const domain = mapSdkSessionToDomain(sdk)
      expect(domain.sdkRestartCount).toBe(0)
    })
  })
})

describe('mapSdkProjectToDomain', () => {
  it('maps SDK project fields to domain project', () => {
    const sdk = makeSdkProject()
    const domain = mapSdkProjectToDomain(sdk)

    expect(domain.id).toBe('proj-001')
    expect(domain.name).toBe('test-project')
    expect(domain.description).toBe('A test project')
    expect(domain.status).toBe('active')
    expect(domain.createdAt).toBe('2026-01-10T08:00:00Z')
    expect(domain.updatedAt).toBe('2026-01-10T09:00:00Z')
  })

  it('maps empty description to null', () => {
    const sdk = makeSdkProject({ description: '' })
    const domain = mapSdkProjectToDomain(sdk)
    expect(domain.description).toBeNull()
  })

  it('maps empty status to null', () => {
    const sdk = makeSdkProject({ status: '' })
    const domain = mapSdkProjectToDomain(sdk)
    expect(domain.status).toBeNull()
  })

  it('handles null created_at and updated_at', () => {
    const sdk = makeSdkProject({
      created_at: null,
      updated_at: null,
    })
    const domain = mapSdkProjectToDomain(sdk)
    expect(domain.createdAt).toBe('')
    expect(domain.updatedAt).toBe('')
  })
})

function makeSdkMessage(overrides: Partial<SdkSessionMessageShape> = {}): SdkSessionMessageShape {
  return {
    id: 'msg-001',
    session_id: 'sess-001',
    event_type: 'tool_use',
    payload: 'git status',
    seq: 1,
    created_at: '2026-01-15T10:00:00Z',
    ...overrides,
  }
}

describe('mapSessionMessageToDomain', () => {
  it('maps snake_case fields to camelCase', () => {
    const sdk = makeSdkMessage()
    const domain = mapSessionMessageToDomain(sdk)

    expect(domain.id).toBe('msg-001')
    expect(domain.sessionId).toBe('sess-001')
    expect(domain.eventType).toBe('tool_use')
    expect(domain.payload).toBe('git status')
    expect(domain.seq).toBe(1)
    expect(domain.createdAt).toBe('2026-01-15T10:00:00Z')
  })

  it('handles null created_at by mapping to empty string', () => {
    const sdk = makeSdkMessage({ created_at: null })
    const domain = mapSessionMessageToDomain(sdk)
    expect(domain.createdAt).toBe('')
  })

  it('preserves payload content exactly', () => {
    const complexPayload = '{"tool":"bash","args":{"command":"ls -la"}}'
    const sdk = makeSdkMessage({ payload: complexPayload })
    const domain = mapSessionMessageToDomain(sdk)
    expect(domain.payload).toBe(complexPayload)
  })
})
