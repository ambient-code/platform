import type { DomainRoleBinding } from '@/domain/types'

// ---------------------------------------------------------------------------
// Indexed lookups – O(1) per cell instead of O(bindings)
// ---------------------------------------------------------------------------

export type BindingIndex = {
  byProject: Map<string, DomainRoleBinding> // key: `${credentialId}:${projectId}`
  byAgent: Map<string, DomainRoleBinding> // key: `${credentialId}:${agentId}`
}

export function buildBindingIndex(bindings: DomainRoleBinding[]): BindingIndex {
  const byProject = new Map<string, DomainRoleBinding>()
  const byAgent = new Map<string, DomainRoleBinding>()
  for (const b of bindings) {
    if (b.credentialId && b.projectId && !b.agentId) {
      byProject.set(`${b.credentialId}:${b.projectId}`, b)
    }
    if (b.credentialId && b.agentId) {
      byAgent.set(`${b.credentialId}:${b.agentId}`, b)
    }
  }
  return { byProject, byAgent }
}

export function findProjectBindingIndexed(
  index: BindingIndex,
  credentialId: string,
  projectId: string,
): DomainRoleBinding | undefined {
  return index.byProject.get(`${credentialId}:${projectId}`)
}

export function findAgentBindingIndexed(
  index: BindingIndex,
  credentialId: string,
  agentId: string,
): DomainRoleBinding | undefined {
  return index.byAgent.get(`${credentialId}:${agentId}`)
}

export function isInheritedIndexed(
  index: BindingIndex,
  credentialId: string,
  agentId: string,
  projectId: string,
): boolean {
  return (
    !!findProjectBindingIndexed(index, credentialId, projectId) &&
    !findAgentBindingIndexed(index, credentialId, agentId)
  )
}

// ---------------------------------------------------------------------------
// Linear-scan lookups – retained for unit tests and non-hot-path usage
// ---------------------------------------------------------------------------

export function cellKey(credentialId: string, targetId: string): string {
  return `${credentialId}:${targetId}`
}

export function findProjectBinding(
  bindings: DomainRoleBinding[],
  credentialId: string,
  projectId: string,
): DomainRoleBinding | undefined {
  return bindings.find(
    (b) =>
      b.credentialId === credentialId &&
      b.projectId === projectId &&
      !b.agentId,
  )
}

export function findAgentBinding(
  bindings: DomainRoleBinding[],
  credentialId: string,
  agentId: string,
): DomainRoleBinding | undefined {
  return bindings.find(
    (b) => b.credentialId === credentialId && b.agentId === agentId,
  )
}

export function isInherited(
  bindings: DomainRoleBinding[],
  credentialId: string,
  agentId: string,
  projectId: string,
): boolean {
  return (
    !!findProjectBinding(bindings, credentialId, projectId) &&
    !findAgentBinding(bindings, credentialId, agentId)
  )
}

export type CellState = 'unbound' | 'project-bound' | 'agent-bound' | 'inherited' | 'both'

export function getCellState(
  bindings: DomainRoleBinding[],
  credentialId: string,
  targetType: 'project' | 'agent',
  targetId: string,
  projectId: string,
): CellState {
  if (targetType === 'project') {
    return findProjectBinding(bindings, credentialId, targetId) ? 'project-bound' : 'unbound'
  }
  const projectBound = !!findProjectBinding(bindings, credentialId, projectId)
  const agentBound = !!findAgentBinding(bindings, credentialId, targetId)
  if (projectBound && agentBound) return 'both'
  if (projectBound && !agentBound) return 'inherited'
  if (!projectBound && agentBound) return 'agent-bound'
  return 'unbound'
}
