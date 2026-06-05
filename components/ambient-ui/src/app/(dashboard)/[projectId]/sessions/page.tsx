'use client'

import { useState, useCallback } from 'react'
import { useParams } from 'next/navigation'
import { Monitor, Plus, FlaskConical } from 'lucide-react'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { EmptyState } from '@/components/empty-state'
import { useSessions } from '@/queries/use-sessions'
import { useAgentNames } from '@/queries/use-agents'
import type { SessionPhase } from '@/domain/types'
import { FleetTable } from './_components/fleet-table'
import { FleetSummary } from './_components/fleet-summary'
import { CreateSessionSheet } from './_components/create-session-sheet'

export default function FleetPage() {
  const { projectId } = useParams<{ projectId: string }>()
  const [search, setSearch] = useState('')
  const [phaseFilter, setPhaseFilter] = useState<SessionPhase | null>(null)
  const [filteredCount, setFilteredCount] = useState<number | undefined>(undefined)
  const [createOpen, setCreateOpen] = useState(false)
  const [showTestRuns, setShowTestRuns] = useState(false)
  const { data, isLoading, error } = useSessions(projectId)
  const { data: agentNames } = useAgentNames(projectId)

  const handleFilteredCountChange = useCallback((count: number) => {
    setFilteredCount(count)
  }, [])

  if (error) {
    return (
      <div className="space-y-6">
        <h1 className="text-2xl font-semibold tracking-tight">Sessions</h1>
        <p className="text-sm text-destructive">
          Failed to load sessions: {error.message}
        </p>
      </div>
    )
  }

  if (isLoading) {
    return (
      <div className="space-y-6">
        <h1 className="text-2xl font-semibold tracking-tight">Sessions</h1>
        <div className="space-y-3">
          <Skeleton className="h-8 w-64" />
          <Skeleton className="h-[400px] w-full" />
        </div>
      </div>
    )
  }

  const sessions = data?.items ?? []
  const testSessionCount = sessions.filter(
    (s) => s.annotations['ambient-code.io/ui/test-session'] === 'true',
  ).length

  if (sessions.length === 0) {
    return (
      <div className="space-y-6">
        <h1 className="text-2xl font-semibold tracking-tight">Sessions</h1>
        <EmptyState
          icon={Monitor}
          title="No sessions"
          description="This project has no agentic sessions yet. Create one to get started."
          action={
            <Button onClick={() => setCreateOpen(true)}>
              <Plus className="mr-1.5 size-4" />
              New Session
            </Button>
          }
        />
        <CreateSessionSheet open={createOpen} onOpenChange={setCreateOpen} />
      </div>
    )
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold tracking-tight">Sessions</h1>
        <div className="flex items-center gap-2">
          {testSessionCount > 0 && (
            <Button
              variant={showTestRuns ? 'secondary' : 'ghost'}
              size="sm"
              onClick={() => setShowTestRuns((prev) => !prev)}
              className="text-xs text-muted-foreground"
            >
              <FlaskConical className="mr-1 size-3.5" />
              {showTestRuns
                ? 'Hide test runs'
                : `Show test runs (${testSessionCount})`}
            </Button>
          )}
          <Input
            placeholder="Filter by name, agent, or model..."
            value={search}
            onChange={e => setSearch(e.target.value)}
            className="max-w-xs"
          />
          <Button onClick={() => setCreateOpen(true)} size="sm">
            <Plus className="mr-1.5 size-4" />
            New Session
          </Button>
        </div>
      </div>
      <FleetSummary
        sessions={sessions}
        filteredCount={filteredCount}
        activePhase={phaseFilter}
        onPhaseFilter={setPhaseFilter}
      />
      <FleetTable
        sessions={sessions}
        searchFilter={search}
        agentNames={agentNames}
        phaseFilter={phaseFilter}
        showTestRuns={showTestRuns}
        onFilteredCountChange={handleFilteredCountChange}
      />
      <CreateSessionSheet open={createOpen} onOpenChange={setCreateOpen} />
    </div>
  )
}
