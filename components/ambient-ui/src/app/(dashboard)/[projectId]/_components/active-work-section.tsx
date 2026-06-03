import Link from 'next/link'
import { Ticket, GitPullRequest, Monitor } from 'lucide-react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { PhaseBadge } from '../sessions/_components/phase-badge'
import type { DomainSession } from '@/domain/types'
import type { WorkItemGroup } from './dashboard-helpers'

type ActiveWorkSectionProps = {
  grouped: WorkItemGroup[]
  ungrouped: DomainSession[]
  projectId: string
}

const REF_TYPE_CONFIG = {
  jira: { icon: Ticket, label: 'Jira' },
  'github-pr': { icon: GitPullRequest, label: 'PR' },
} as const

function WorkItemCard({
  group,
  projectId,
}: {
  group: WorkItemGroup
  projectId: string
}) {
  const config = REF_TYPE_CONFIG[group.ref.type]
  const Icon = config.icon

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2 text-sm">
          <Icon className="size-4 text-muted-foreground" />
          <span>{group.ref.key}</span>
          <Badge variant="outline" className="ml-auto text-xs">
            {config.label}
          </Badge>
        </CardTitle>
      </CardHeader>
      <CardContent>
        <ul className="space-y-1.5">
          {group.sessions.map(session => (
            <li key={session.id}>
              <Link
                href={`/${projectId}/sessions/${session.id}`}
                className="flex items-center gap-2 text-sm"
              >
                <PhaseBadge phase={session.phase} />
                <span className="truncate text-link hover:text-link-hover">
                  {session.name}
                </span>
                {session.agentName && (
                  <span className="ml-auto shrink-0 text-xs text-muted-foreground">
                    {session.agentName}
                  </span>
                )}
              </Link>
            </li>
          ))}
        </ul>
      </CardContent>
    </Card>
  )
}

function SessionCard({
  session,
  projectId,
}: {
  session: DomainSession
  projectId: string
}) {
  return (
    <Card className="py-4">
      <CardContent className="flex items-center gap-3">
        <Monitor className="size-4 shrink-0 text-muted-foreground" />
        <Link
          href={`/${projectId}/sessions/${session.id}`}
          className="truncate text-sm font-medium text-link hover:text-link-hover"
        >
          {session.name}
        </Link>
        <PhaseBadge phase={session.phase} />
        {session.agentName && (
          <span className="ml-auto shrink-0 text-xs text-muted-foreground">
            {session.agentName}
          </span>
        )}
      </CardContent>
    </Card>
  )
}

export function ActiveWorkSection({ grouped, ungrouped, projectId }: ActiveWorkSectionProps) {
  const hasWork = grouped.length > 0 || ungrouped.length > 0

  if (!hasWork) {
    return (
      <div>
        <h2 className="mb-3 text-sm font-semibold">Active work</h2>
        <p className="text-sm text-muted-foreground">
          No sessions are currently running.
        </p>
      </div>
    )
  }

  return (
    <div>
      <h2 className="mb-3 text-sm font-semibold">Active work</h2>
      <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
        {grouped.map(group => (
          <WorkItemCard
            key={`${group.ref.type}:${group.ref.key}`}
            group={group}
            projectId={projectId}
          />
        ))}
        {ungrouped.map(session => (
          <SessionCard key={session.id} session={session} projectId={projectId} />
        ))}
      </div>
    </div>
  )
}
