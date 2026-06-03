import type { DomainSession, SessionPhase } from '@/domain/types'
import { PhaseBadge } from './phase-badge'

export function FleetSummary({ sessions }: { sessions: DomainSession[] }) {
  const counts = sessions.reduce<Partial<Record<SessionPhase, number>>>((acc, s) => {
    acc[s.phase] = (acc[s.phase] ?? 0) + 1
    return acc
  }, {})

  const phases: SessionPhase[] = ['Running', 'Pending', 'Creating', 'Stopping', 'Failed', 'Completed', 'Stopped']

  return (
    <div className="flex items-center gap-4 text-sm">
      <span className="font-medium">{sessions.length} sessions</span>
      <span className="text-muted-foreground">—</span>
      {phases.map(phase => {
        const count = counts[phase]
        if (!count) return null
        return (
          <div key={phase} className="flex items-center gap-1.5">
            <PhaseBadge phase={phase} />
            <span className="text-muted-foreground">{count}</span>
          </div>
        )
      })}
    </div>
  )
}
