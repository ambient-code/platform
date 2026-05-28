import type { DomainSession } from '@/domain/types'
import { PhaseBadge } from '../../_components/phase-badge'
import { formatDuration, formatRelativeTime } from '@/lib/format-timestamp'

export function SessionHeader({ session }: { session: DomainSession }) {
  return (
    <div className="space-y-3">
      <div className="flex items-center gap-3">
        <h1 className="text-2xl font-semibold">{session.name}</h1>
        <PhaseBadge phase={session.phase} />
      </div>
      <div className="flex items-center gap-6 text-sm text-muted-foreground">
        {session.agentName && (
          <MetaItem label="Agent" value={session.agentName} />
        )}
        {session.model && (
          <MetaItem label="Model" value={session.model} />
        )}
        {session.startTime && (
          <MetaItem
            label="Duration"
            value={formatDuration(session.startTime, session.completionTime)}
          />
        )}
        <MetaItem label="Created" value={formatRelativeTime(session.createdAt)} />
      </div>
    </div>
  )
}

function MetaItem({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <span className="text-muted-foreground/70">{label}:</span>{' '}
      <span>{value}</span>
    </div>
  )
}
