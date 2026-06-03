'use client'

import { useState } from 'react'
import { useParams } from 'next/navigation'
import { Bot } from 'lucide-react'
import { Input } from '@/components/ui/input'
import { Skeleton } from '@/components/ui/skeleton'
import { EmptyState } from '@/components/empty-state'
import { useAgents } from '@/queries/use-agents'
import type { DomainAgent } from '@/domain/types'
import { AgentsTable } from './_components/agents-table'
import { AgentDetailPanel } from './_components/agent-detail-panel'

export default function AgentsPage() {
  const { projectId } = useParams<{ projectId: string }>()
  const [search, setSearch] = useState('')
  const [selectedAgent, setSelectedAgent] = useState<DomainAgent | null>(null)
  const { data, isLoading, error } = useAgents(projectId)

  if (error) {
    return (
      <div className="space-y-6">
        <h1 className="text-xl font-semibold">Agents</h1>
        <p className="text-sm text-destructive">
          Failed to load agents: {error.message}
        </p>
      </div>
    )
  }

  if (isLoading) {
    return (
      <div className="space-y-6">
        <h1 className="text-xl font-semibold">Agents</h1>
        <div className="space-y-3">
          <Skeleton className="h-8 w-64" />
          <Skeleton className="h-[400px] w-full" />
        </div>
      </div>
    )
  }

  const agents = data?.items ?? []

  if (agents.length === 0) {
    return (
      <div className="space-y-6">
        <h1 className="text-xl font-semibold">Agents</h1>
        <EmptyState
          icon={Bot}
          title="No agents"
          description="This project has no agents yet."
        />
      </div>
    )
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-semibold">Agents</h1>
        <Input
          placeholder="Filter by name, model, or owner..."
          value={search}
          onChange={e => setSearch(e.target.value)}
          className="max-w-xs"
        />
      </div>
      <AgentsTable
        agents={agents}
        searchFilter={search}
        onSelectAgent={setSelectedAgent}
      />
      <AgentDetailPanel
        agent={selectedAgent}
        projectId={projectId}
        onClose={() => setSelectedAgent(null)}
      />
    </div>
  )
}
