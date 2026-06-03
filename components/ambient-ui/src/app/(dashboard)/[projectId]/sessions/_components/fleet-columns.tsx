import { createColumnHelper } from '@tanstack/react-table'
import type { SortingFn } from '@tanstack/react-table'
import { MessageSquare } from 'lucide-react'
import { Button } from '@/components/ui/button'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import type { DomainSession, SessionPhase } from '@/domain/types'
import { formatRelativeTime, formatPreciseDuration } from '@/lib/format-timestamp'
import { useChatSidebar } from '@/components/chat-sidebar-context'
import { PhaseBadge } from './phase-badge'

const COST_ANNOTATION = 'ambient-code.io/cost/estimate'
const col = createColumnHelper<DomainSession>()

const RUNNING_PHASES: ReadonlySet<SessionPhase> = new Set(['Running', 'Creating', 'Pending', 'Stopping'])
const TERMINAL_PHASES: ReadonlySet<SessionPhase> = new Set(['Completed', 'Failed', 'Stopped'])

/** Priority order for phase sorting: Failed first (0), terminal last */
const PHASE_SORT_PRIORITY: Record<SessionPhase, number> = {
  Failed: 0,
  Running: 1,
  Stopping: 2,
  Creating: 3,
  Pending: 4,
  Completed: 5,
  Stopped: 6,
}

const phaseSortingFn: SortingFn<DomainSession> = (rowA, rowB) => {
  const a = PHASE_SORT_PRIORITY[rowA.original.phase] ?? 99
  const b = PHASE_SORT_PRIORITY[rowB.original.phase] ?? 99
  return a - b
}

function ChatColumnButton({ sessionId, phase }: { sessionId: string; phase: SessionPhase }) {
  const { openSidebar, openSessionId } = useChatSidebar()
  const isActive = openSessionId === sessionId
  const isTerminal = TERMINAL_PHASES.has(phase)

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <Button
          variant="ghost"
          size="icon"
          className="h-7 w-7"
          onClick={(e) => {
            e.stopPropagation()
            openSidebar(sessionId)
          }}
          aria-label={isTerminal ? 'View chat history' : 'Open chat sidebar'}
        >
          <MessageSquare
            className={`h-4 w-4 ${isActive ? 'text-primary' : 'text-muted-foreground'}`}
          />
        </Button>
      </TooltipTrigger>
      <TooltipContent>
        {isActive
          ? 'Chat sidebar is open'
          : isTerminal
            ? 'View chat history'
            : 'Open chat in sidebar'}
      </TooltipContent>
    </Tooltip>
  )
}

export const fleetColumns = [
  col.accessor('phase', {
    header: 'Phase',
    cell: info => <PhaseBadge phase={info.getValue()} />,
    size: 130,
    enableSorting: true,
    sortingFn: phaseSortingFn,
  }),
  col.accessor('name', {
    header: 'Name',
    cell: info => (
      <span className="font-medium">{info.getValue()}</span>
    ),
  }),
  col.accessor('agentId', {
    header: 'Agent',
    cell: info => {
      const agentId = info.getValue()
      const annotationName = info.row.original.agentName
      const agentNames = (info.table.options.meta as { agentNames?: Map<string, string> } | undefined)?.agentNames
      const resolvedName = annotationName ?? (agentId ? agentNames?.get(agentId) : null) ?? null
      if (!resolvedName) return <span className="text-muted-foreground">—</span>
      return (
        <span className="text-sm text-foreground">
          {resolvedName}
        </span>
      )
    },
  }),
  col.display({
    id: 'duration',
    header: 'Duration',
    enableSorting: true,
    sortingFn: (rowA, rowB) => {
      const getMs = (row: typeof rowA) => {
        const { startTime, completionTime, phase } = row.original
        if (!startTime) return 0
        const isActive = RUNNING_PHASES.has(phase)
        const end = isActive ? new Date() : (completionTime ? new Date(completionTime) : new Date())
        return Math.max(0, end.getTime() - new Date(startTime).getTime())
      }
      return getMs(rowA) - getMs(rowB)
    },
    cell: ({ row }) => {
      const { startTime, completionTime, phase } = row.original
      if (!startTime) return <span className="text-muted-foreground">—</span>
      const isActive = RUNNING_PHASES.has(phase)
      const endTime = isActive ? null
        : (completionTime && new Date(completionTime) > new Date(startTime)) ? completionTime
        : null
      return (
        <span className="text-muted-foreground font-mono text-xs">
          {formatPreciseDuration(startTime, endTime)}
        </span>
      )
    },
  }),
  col.accessor('model', {
    header: 'Model',
    cell: info => (
      <span className="text-muted-foreground text-xs">
        {info.getValue() ?? '—'}
      </span>
    ),
  }),
  col.display({
    id: 'lastActivity',
    header: 'Last Activity',
    enableSorting: true,
    sortingFn: (rowA, rowB) => {
      const getTime = (row: typeof rowA) => {
        const { phase, completionTime, updatedAt } = row.original
        if (RUNNING_PHASES.has(phase)) return Date.now()
        return new Date(completionTime ?? updatedAt).getTime()
      }
      return getTime(rowA) - getTime(rowB)
    },
    cell: ({ row }) => {
      const { phase, completionTime, updatedAt } = row.original

      if (phase === 'Running') {
        return (
          <span className="text-xs font-medium text-status-success-foreground">
            Active now
          </span>
        )
      }

      if (phase === 'Creating' || phase === 'Pending') {
        return (
          <span className="text-xs font-medium text-status-warning-foreground">
            Starting...
          </span>
        )
      }

      if (phase === 'Stopping') {
        return (
          <span className="text-xs font-medium text-muted-foreground">
            Stopping...
          </span>
        )
      }

      const activityTime = completionTime ?? updatedAt
      return (
        <span className="text-muted-foreground text-xs">
          {formatRelativeTime(activityTime)}
        </span>
      )
    },
  }),
  col.display({
    id: 'cost',
    header: 'Cost',
    enableSorting: true,
    sortingFn: (rowA, rowB) => {
      const getCost = (row: typeof rowA) => {
        const raw = row.original.annotations[COST_ANNOTATION]
        if (!raw) return 0
        return parseFloat(raw.replace(/[^0-9.]/g, '')) || 0
      }
      return getCost(rowA) - getCost(rowB)
    },
    cell: ({ row }) => {
      const cost = row.original.annotations[COST_ANNOTATION]
      return (
        <span className="text-muted-foreground text-xs font-mono">
          {cost ?? '—'}
        </span>
      )
    },
    size: 80,
  }),
  col.display({
    id: 'chat',
    header: () => (
      <MessageSquare className="size-3.5 text-muted-foreground" aria-label="Chat" />
    ),
    cell: ({ row }) => <ChatColumnButton sessionId={row.original.id} phase={row.original.phase} />,
    size: 48,
    enableSorting: false,
  }),
]
