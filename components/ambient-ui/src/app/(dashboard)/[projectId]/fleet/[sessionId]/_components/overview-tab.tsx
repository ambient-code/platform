import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import type { DomainSession, SessionPhase } from '@/domain/types'
import { cn } from '@/lib/utils'
import { formatAbsoluteTime } from '@/lib/format-timestamp'
import { MetaRow, NoValue } from './meta-row'

const LIFECYCLE: SessionPhase[] = ['Pending', 'Creating', 'Running']

const TERMINAL_ORDER = 4

const PHASE_ORDER: Record<SessionPhase, number> = {
  Pending: 0, Creating: 1, Running: 2, Stopping: 3,
  Completed: TERMINAL_ORDER, Failed: TERMINAL_ORDER, Stopped: TERMINAL_ORDER,
}

function phaseColor(phase: SessionPhase): string {
  switch (phase) {
    case 'Running':
      return 'bg-green-500 border-green-500'
    case 'Failed':
      return 'bg-red-500 border-red-500'
    case 'Completed':
      return 'bg-blue-500 border-blue-500'
    case 'Stopped':
      return 'bg-muted-foreground border-muted-foreground'
    default:
      return 'bg-foreground border-foreground'
  }
}

export function OverviewTab({ session }: { session: DomainSession }) {
  const currentOrder = PHASE_ORDER[session.phase]

  return (
    <div className="space-y-6 pt-4">
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Phase Timeline</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex items-center gap-2">
            {LIFECYCLE.map((phase, i) => {
              const order = PHASE_ORDER[phase]
              const isCurrent = phase === session.phase
              const isPast = order < currentOrder
              return (
                <div key={phase} className="flex items-center gap-2">
                  {i > 0 && (
                    <div className={cn(
                      'h-0.5 w-8',
                      isPast || isCurrent ? 'bg-foreground' : 'bg-border',
                    )} />
                  )}
                  <div className="flex flex-col items-center gap-1">
                    <div className={cn(
                      'h-3 w-3 rounded-full border-2',
                      isCurrent && phaseColor(session.phase),
                      isPast && 'bg-foreground border-foreground',
                      !isCurrent && !isPast && 'bg-background border-muted-foreground/40',
                    )} />
                    <span className={cn(
                      'text-xs',
                      isCurrent ? 'font-medium' : 'text-muted-foreground',
                    )}>
                      {phase}
                    </span>
                  </div>
                </div>
              )
            })}
            <div className="h-0.5 w-8 bg-border" />
            <div className="flex flex-col items-center gap-1">
              <div className={cn(
                'h-3 w-3 rounded-full border-2',
                currentOrder >= TERMINAL_ORDER
                  ? phaseColor(session.phase)
                  : 'bg-background border-muted-foreground/40',
              )} />
              <span className={cn(
                'text-xs',
                currentOrder >= TERMINAL_ORDER ? 'font-medium' : 'text-muted-foreground',
              )}>
                {currentOrder >= TERMINAL_ORDER ? session.phase : 'Terminal'}
              </span>
            </div>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">Timing</CardTitle>
        </CardHeader>
        <CardContent>
          <dl className="grid grid-cols-1 sm:grid-cols-2 gap-x-8 gap-y-3 text-sm">
            <MetaRow label="Session ID" value={session.id} mono />
            <MetaRow label="Project" value={session.projectId ?? <NoValue />} />
            <MetaRow label="Agent" value={session.agentName ?? session.agentId ?? <NoValue />} />
            <MetaRow label="Started" value={session.startTime ? formatAbsoluteTime(session.startTime) : <NoValue />} />
            <MetaRow label="Completed" value={session.completionTime ? formatAbsoluteTime(session.completionTime) : <NoValue />} />
          </dl>
        </CardContent>
      </Card>
    </div>
  )
}
