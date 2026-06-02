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
