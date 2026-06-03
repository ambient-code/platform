'use client'

import { GitBranch, Pencil } from 'lucide-react'
import { Badge } from '@/components/ui/badge'

const MANAGED_BY_KEY = 'ambient-code.io/managed-by'

export type AgentLifecycle = 'draft' | 'gitops'

export function getAgentLifecycle(annotations: Record<string, string>): AgentLifecycle {
  return annotations[MANAGED_BY_KEY] === 'gitops' ? 'gitops' : 'draft'
}

export function LifecycleBadge({ lifecycle }: { lifecycle: AgentLifecycle }) {
  if (lifecycle === 'gitops') {
    return (
      <Badge variant="secondary" className="gap-1 text-muted-foreground">
        <GitBranch className="size-3" />
        GitOps
      </Badge>
    )
  }

  return (
    <Badge variant="outline" className="gap-1">
      <Pencil className="size-3" />
      Draft
    </Badge>
  )
}
