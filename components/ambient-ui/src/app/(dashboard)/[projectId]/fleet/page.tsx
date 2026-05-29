'use client'

import { useState } from 'react'
import { useParams } from 'next/navigation'
import { Monitor } from 'lucide-react'
import { Input } from '@/components/ui/input'
import { Skeleton } from '@/components/ui/skeleton'
import { EmptyState } from '@/components/empty-state'
import { useSessions } from '@/queries/use-sessions'
import { FleetTable } from './_components/fleet-table'
import { FleetSummary } from './_components/fleet-summary'

export default function FleetPage() {
  const { projectId } = useParams<{ projectId: string }>()
  const [search, setSearch] = useState('')
  const { data, isLoading, error } = useSessions(projectId)

  if (error) {
    return (
      <div className="space-y-6">
        <h1 className="text-2xl font-semibold">Fleet</h1>
        <p className="text-sm text-destructive">
          Failed to load sessions: {error.message}
        </p>
      </div>
    )
  }

  if (isLoading) {
    return (
      <div className="space-y-6">
        <h1 className="text-2xl font-semibold">Fleet</h1>
        <div className="space-y-3">
          <Skeleton className="h-8 w-64" />
          <Skeleton className="h-[400px] w-full" />
        </div>
      </div>
    )
  }

  const sessions = data?.items ?? []

  if (sessions.length === 0) {
    return (
      <div className="space-y-6">
        <h1 className="text-2xl font-semibold">Fleet</h1>
        <EmptyState
          icon={Monitor}
          title="No sessions"
          description="This project has no agentic sessions yet. Create one to get started."
        />
      </div>
    )
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold">Fleet</h1>
        <Input
          placeholder="Filter sessions..."
          value={search}
          onChange={e => setSearch(e.target.value)}
          className="max-w-xs"
        />
      </div>
      <FleetSummary sessions={sessions} />
      <FleetTable sessions={sessions} searchFilter={search} />
    </div>
  )
}
