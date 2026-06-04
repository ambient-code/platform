import type { DomainRoleBinding } from '@/domain/types'

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
