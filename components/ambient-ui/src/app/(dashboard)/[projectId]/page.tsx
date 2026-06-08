'use client'

import { useParams } from 'next/navigation'
import { LayoutDashboard } from 'lucide-react'
import { Skeleton } from '@/components/ui/skeleton'
import { EmptyState } from '@/components/empty-state'
import { useSessions } from '@/queries/use-sessions'
import { AttentionBanner } from './_components/attention-banner'
import { ActiveWorkSection } from './_components/active-work-section'
import { RecentActivity } from './_components/recent-activity'
import {
  getAttentionItems,
  getActiveWorkItems,
  getRecentActivity,
} from './_components/dashboard-helpers'

export default function DashboardPage() {
  const { projectId } = useParams<{ projectId: string }>()
  const { data, isLoading, error } = useSessions(projectId)

  if (error) {
    return (
      <div className="space-y-6">
        <h1 className="text-2xl font-semibold tracking-tight">Dashboard</h1>
        <p className="text-sm text-destructive">
          Failed to load dashboard data. Please try again later.
        </p>
      </div>
    )
  }

  if (isLoading) {
    return (
      <div className="space-y-6">
        <h1 className="text-2xl font-semibold tracking-tight">Dashboard</h1>
        <Skeleton className="h-24 w-full" />
        <Skeleton className="h-48 w-full" />
        <Skeleton className="h-64 w-full" />
      </div>
    )
  }

  const sessions = data?.items ?? []

  if (sessions.length === 0) {
    return (
      <div className="space-y-6">
        <h1 className="text-2xl font-semibold tracking-tight">Dashboard</h1>
        <EmptyState
          icon={LayoutDashboard}
          title="No sessions yet"
          description="Create a session from the Sessions page to see your dashboard come to life."
        />
      </div>
    )
  }

  const attentionItems = getAttentionItems(sessions)
  const { grouped, ungrouped } = getActiveWorkItems(sessions)
  const recentItems = getRecentActivity(sessions)

  return (
    <div className="@container space-y-6">
      <h1 className="text-2xl font-semibold tracking-tight">Dashboard</h1>
      <AttentionBanner items={attentionItems} projectId={projectId} />
      <ActiveWorkSection
        grouped={grouped}
        ungrouped={ungrouped}
        projectId={projectId}
      />
      <RecentActivity items={recentItems} projectId={projectId} />
    </div>
  )
}
