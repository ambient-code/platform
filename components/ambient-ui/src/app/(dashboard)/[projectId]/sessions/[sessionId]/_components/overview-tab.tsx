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

function TimelineSteps({ session, currentOrder }: { session: DomainSession; currentOrder: number }) {
  const terminalLabel = currentOrder >= TERMINAL_ORDER ? session.phase : 'Terminal'
  const terminalActive = currentOrder >= TERMINAL_ORDER
  const steps = [
    ...LIFECYCLE.map((phase) => ({
      label: phase,
      isCurrent: phase === session.phase,
      isPast: PHASE_ORDER[phase] < currentOrder,
    })),
    { label: terminalLabel, isCurrent: terminalActive, isPast: false },
  ]

  return (
    <div className="flex items-start">
      {steps.map((step, i) => (
        <div key={step.label} className="flex items-start">
          {i > 0 && (
            <div className={cn(
              'mt-[5px] h-0.5 w-8',
              (step.isPast || step.isCurrent) ? 'bg-foreground' : 'bg-border',
            )} />
          )}
          <div className="flex flex-col items-center gap-1">
            <div className={cn(
              'size-3 shrink-0 rounded-full border-2',
              step.isCurrent && phaseColor(session.phase),
              step.isPast && 'bg-foreground border-foreground',
              !step.isCurrent && !step.isPast && 'bg-background border-muted-foreground/40',
            )} />
            <span className={cn(
              'text-xs',
              step.isCurrent ? 'font-medium' : 'text-muted-foreground',
            )}>
              {step.label}
            </span>
          </div>
        </div>
      ))}
    </div>
  )
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
          <TimelineSteps session={session} currentOrder={currentOrder} />
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
