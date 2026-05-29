import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import type { DomainSession, SessionPhase } from '@/domain/types'
import { cn } from '@/lib/utils'
import { formatAbsoluteTime } from '@/lib/format-timestamp'

const LIFECYCLE: SessionPhase[] = ['Pending', 'Creating', 'Running']

const TERMINAL_ORDER = 4

const PHASE_ORDER: Record<SessionPhase, number> = {
  Pending: 0, Creating: 1, Running: 2, Stopping: 3,
  Completed: TERMINAL_ORDER, Failed: TERMINAL_ORDER, Stopped: TERMINAL_ORDER,
}

export function PhaseTab({ session }: { session: DomainSession }) {
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
                      isCurrent && 'bg-foreground border-foreground',
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
                currentOrder >= TERMINAL_ORDER ? 'bg-foreground border-foreground' : 'bg-background border-muted-foreground/40',
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
          <CardTitle className="text-base">Metadata</CardTitle>
        </CardHeader>
        <CardContent>
          <dl className="grid grid-cols-2 gap-x-8 gap-y-3 text-sm">
            <MetaRow label="Session ID" value={session.id} mono />
            <MetaRow label="Project" value={session.projectId ?? '—'} />
            <MetaRow label="Agent" value={session.agentName ?? session.agentId ?? '—'} />
            <MetaRow label="Model" value={session.model ?? '—'} />
            <MetaRow label="Started" value={session.startTime ? formatAbsoluteTime(session.startTime) : '—'} />
            <MetaRow label="Completed" value={session.completionTime ? formatAbsoluteTime(session.completionTime) : '—'} />
          </dl>
        </CardContent>
      </Card>

      {Object.keys(session.annotations).length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Annotations</CardTitle>
          </CardHeader>
          <CardContent>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Key</TableHead>
                  <TableHead>Value</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {Object.entries(session.annotations).map(([key, value]) => (
                  <TableRow key={key}>
                    <TableCell className="font-mono text-xs">{key}</TableCell>
                    <TableCell className="text-sm">{value}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      )}
    </div>
  )
}

function MetaRow({ label, value, mono }: { label: string; value: string; mono?: boolean }) {
  return (
    <div>
      <dt className="text-muted-foreground">{label}</dt>
      <dd className={cn('mt-0.5', mono && 'font-mono text-xs')}>{value}</dd>
    </div>
  )
}
