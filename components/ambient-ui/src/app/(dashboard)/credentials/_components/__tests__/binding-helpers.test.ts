import { describe, it, expect } from 'vitest'
import type { DomainRoleBinding } from '@/domain/types'
import {
  cellKey,
  findProjectBinding,
  findAgentBinding,
  isInherited,
  getCellState,
} from '../binding-helpers'

function makeBinding(overrides: Partial<DomainRoleBinding> = {}): DomainRoleBinding {
  return {
    id: 'rb-1',
    roleId: 'role-1',
    scope: 'credential',
    userId: null,
    projectId: null,
    agentId: null,
    credentialId: null,
    sessionId: null,
    createdAt: '2026-01-01T00:00:00Z',
    updatedAt: '2026-01-01T00:00:00Z',
    ...overrides,
  }
}

describe('cellKey', () => {
  it('joins credential and target IDs', () => {
    expect(cellKey('cred-1', 'proj-1')).toBe('cred-1:proj-1')
  })

  it('produces distinct keys for different pairs', () => {
    expect(cellKey('a', 'b')).not.toBe(cellKey('b', 'a'))
  })
})

describe('findProjectBinding', () => {
  const projectBinding = makeBinding({
    id: 'rb-proj',
    credentialId: 'cred-1',
    projectId: 'proj-1',
    agentId: null,
  })

  const agentBinding = makeBinding({
    id: 'rb-agent',
    credentialId: 'cred-1',
    projectId: 'proj-1',
    agentId: 'agent-1',
  })

  it('finds a project-level binding (no agentId)', () => {
    expect(findProjectBinding([projectBinding], 'cred-1', 'proj-1')).toBe(projectBinding)
  })

  it('ignores bindings that have an agentId', () => {
    expect(findProjectBinding([agentBinding], 'cred-1', 'proj-1')).toBeUndefined()
  })

  it('returns undefined when credential does not match', () => {
    expect(findProjectBinding([projectBinding], 'cred-other', 'proj-1')).toBeUndefined()
  })

  it('returns undefined when project does not match', () => {
    expect(findProjectBinding([projectBinding], 'cred-1', 'proj-other')).toBeUndefined()
  })

  it('returns undefined for empty bindings', () => {
    expect(findProjectBinding([], 'cred-1', 'proj-1')).toBeUndefined()
  })

  it('returns the first match when multiple exist', () => {
    const dup = makeBinding({ id: 'rb-dup', credentialId: 'cred-1', projectId: 'proj-1' })
    expect(findProjectBinding([projectBinding, dup], 'cred-1', 'proj-1')).toBe(projectBinding)
  })
})

describe('findAgentBinding', () => {
  const agentBinding = makeBinding({
    id: 'rb-agent',
    credentialId: 'cred-1',
    agentId: 'agent-1',
  })

  const projectBinding = makeBinding({
    id: 'rb-proj',
    credentialId: 'cred-1',
    projectId: 'proj-1',
  })

  it('finds a binding matching credential + agent', () => {
    expect(findAgentBinding([agentBinding], 'cred-1', 'agent-1')).toBe(agentBinding)
  })

  it('does not match project-only bindings', () => {
    expect(findAgentBinding([projectBinding], 'cred-1', 'agent-1')).toBeUndefined()
  })

  it('returns undefined when agent does not match', () => {
    expect(findAgentBinding([agentBinding], 'cred-1', 'agent-other')).toBeUndefined()
  })

  it('returns undefined when credential does not match', () => {
    expect(findAgentBinding([agentBinding], 'cred-other', 'agent-1')).toBeUndefined()
  })
})

describe('isInherited', () => {
  const projectBinding = makeBinding({
    id: 'rb-proj',
    credentialId: 'cred-1',
    projectId: 'proj-1',
  })

  const agentBinding = makeBinding({
    id: 'rb-agent',
    credentialId: 'cred-1',
    agentId: 'agent-1',
  })

  it('returns true when project is bound but agent is not', () => {
    expect(isInherited([projectBinding], 'cred-1', 'agent-1', 'proj-1')).toBe(true)
  })

  it('returns false when neither project nor agent is bound', () => {
    expect(isInherited([], 'cred-1', 'agent-1', 'proj-1')).toBe(false)
  })

  it('returns false when both project and agent are bound', () => {
    expect(isInherited([projectBinding, agentBinding], 'cred-1', 'agent-1', 'proj-1')).toBe(false)
  })

  it('returns false when only agent is bound (no project binding)', () => {
    expect(isInherited([agentBinding], 'cred-1', 'agent-1', 'proj-1')).toBe(false)
  })

  it('returns false for a different credential', () => {
    expect(isInherited([projectBinding], 'cred-other', 'agent-1', 'proj-1')).toBe(false)
  })

  it('returns false for a different project', () => {
    expect(isInherited([projectBinding], 'cred-1', 'agent-1', 'proj-other')).toBe(false)
  })
})

describe('getCellState', () => {
  const projectBinding = makeBinding({
    id: 'rb-proj',
    credentialId: 'cred-1',
    projectId: 'proj-1',
  })

  const agentBinding = makeBinding({
    id: 'rb-agent',
    credentialId: 'cred-1',
    agentId: 'agent-1',
    projectId: 'proj-1',
  })

  describe('project cells', () => {
    it('returns project-bound when a project binding exists', () => {
      expect(getCellState([projectBinding], 'cred-1', 'project', 'proj-1', 'proj-1'))
        .toBe('project-bound')
    })

    it('returns unbound when no project binding exists', () => {
      expect(getCellState([], 'cred-1', 'project', 'proj-1', 'proj-1'))
        .toBe('unbound')
    })

    it('ignores agent bindings for project cells', () => {
      expect(getCellState([agentBinding], 'cred-1', 'project', 'proj-1', 'proj-1'))
        .toBe('unbound')
    })
  })

  describe('agent cells', () => {
    it('returns inherited when project is bound but agent is not', () => {
      expect(getCellState([projectBinding], 'cred-1', 'agent', 'agent-1', 'proj-1'))
        .toBe('inherited')
    })

    it('returns agent-bound when only agent has a direct binding', () => {
      expect(getCellState([agentBinding], 'cred-1', 'agent', 'agent-1', 'proj-1'))
        .toBe('agent-bound')
    })

    it('returns both when project and agent are both bound', () => {
      expect(getCellState([projectBinding, agentBinding], 'cred-1', 'agent', 'agent-1', 'proj-1'))
        .toBe('both')
    })

    it('returns unbound when neither project nor agent is bound', () => {
      expect(getCellState([], 'cred-1', 'agent', 'agent-1', 'proj-1'))
        .toBe('unbound')
    })
  })

  describe('cross-credential isolation', () => {
    it('does not leak bindings across credentials', () => {
      expect(getCellState([projectBinding], 'cred-other', 'project', 'proj-1', 'proj-1'))
        .toBe('unbound')
      expect(getCellState([projectBinding], 'cred-other', 'agent', 'agent-1', 'proj-1'))
        .toBe('unbound')
    })
  })

  describe('cross-project isolation', () => {
    it('project binding in proj-1 does not affect agent in proj-2', () => {
      expect(getCellState([projectBinding], 'cred-1', 'agent', 'agent-1', 'proj-2'))
        .toBe('unbound')
    })
  })

  describe('lifecycle scenarios', () => {
    it('grant project → agent shows inherited', () => {
      const bindings = [projectBinding]
      expect(getCellState(bindings, 'cred-1', 'agent', 'agent-1', 'proj-1')).toBe('inherited')
    })

    it('grant project → add direct agent binding → shows both', () => {
      const bindings = [projectBinding, agentBinding]
      expect(getCellState(bindings, 'cred-1', 'agent', 'agent-1', 'proj-1')).toBe('both')
    })

    it('revoke project binding → agent direct binding persists → shows agent-bound', () => {
      const bindings = [agentBinding]
      expect(getCellState(bindings, 'cred-1', 'agent', 'agent-1', 'proj-1')).toBe('agent-bound')
    })

    it('revoke agent direct binding → project still bound → shows inherited', () => {
      const bindings = [projectBinding]
      expect(getCellState(bindings, 'cred-1', 'agent', 'agent-1', 'proj-1')).toBe('inherited')
    })

    it('revoke both → shows unbound', () => {
      expect(getCellState([], 'cred-1', 'agent', 'agent-1', 'proj-1')).toBe('unbound')
    })
  })
})
