import Link from 'next/link'
import { Ticket, GitPullRequest } from 'lucide-react'
import { PhaseBadge } from '../sessions/_components/phase-badge'
import { formatRelativeTime, formatPreciseDuration } from '@/lib/format-timestamp'
import type { RecentActivityItem } from './dashboard-helpers'

type RecentActivityProps = {
  items: RecentActivityItem[]
  projectId: string
}

const REF_ICONS = {
  jira: Ticket,
  'github-pr': GitPullRequest,
} as const

export function RecentActivity({ items, projectId }: RecentActivityProps) {
  if (items.length === 0) {
    return (
      <div>
        <h2 className="mb-3 text-sm font-semibold">Recent activity</h2>
        <p className="text-sm text-muted-foreground">
          No completed sessions yet.
        </p>
      </div>
    )
  }

  return (
    <div>
      <h2 className="mb-3 text-sm font-semibold">Recent activity</h2>
      <div className="rounded-lg border">
        <ul className="divide-y">
          {items.map(item => {
            const { session, ref, cost } = item
            const completionTime = session.completionTime ?? session.updatedAt
            const duration = session.startTime
              ? formatPreciseDuration(session.startTime, session.completionTime)
              : null

            return (
              <li key={session.id} className="flex items-center gap-3 px-4 py-2.5">
                <PhaseBadge phase={session.phase} />

                {ref ? (
                  <span className="flex items-center gap-1.5 text-sm">
                    {(() => {
                      const Icon = REF_ICONS[ref.type]
                      return <Icon className="size-3.5 text-muted-foreground" />
                    })()}
                    <span className="font-medium">{ref.key}</span>
                  </span>
                ) : null}

                <Link
                  href={`/${projectId}/sessions/${session.id}`}
                  className="truncate text-sm text-link hover:text-link-hover"
                >
                  {session.name}
                </Link>

                <div className="ml-auto flex shrink-0 items-center gap-3 text-xs text-muted-foreground">
                  {duration && (
                    <span className="font-mono">{duration}</span>
                  )}
                  {cost && (
                    <span className="font-mono">{cost}</span>
                  )}
                  <span>{formatRelativeTime(completionTime)}</span>
                </div>
              </li>
            )
          })}
        </ul>
      </div>
    </div>
  )
}
