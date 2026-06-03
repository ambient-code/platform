import { createColumnHelper } from '@tanstack/react-table'
import { MessageSquare } from 'lucide-react'
import { Button } from '@/components/ui/button'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import type { DomainSession, SessionPhase } from '@/domain/types'
import { formatRelativeTime, formatDuration } from '@/lib/format-timestamp'
import { useChatSidebar } from '@/components/chat-sidebar-context'
import { PhaseBadge } from './phase-badge'

const COST_ANNOTATION = 'ambient-code.io/cost/estimate'
const col = createColumnHelper<DomainSession>()

const RUNNING_PHASES: ReadonlySet<SessionPhase> = new Set(['Running', 'Creating', 'Pending', 'Stopping'])

function ChatColumnButton({ sessionId }: { sessionId: string }) {
  const { openSidebar, openSessionId } = useChatSidebar()
  const isActive = openSessionId === sessionId

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
          aria-label="Open chat sidebar"
        >
          <MessageSquare
            className={`h-4 w-4 ${isActive ? 'text-primary' : 'text-muted-foreground'}`}
          />
        </Button>
      </TooltipTrigger>
      <TooltipContent>
        {isActive ? 'Chat sidebar is open' : 'Open chat in sidebar'}
      </TooltipContent>
    </Tooltip>
  )
}

export const fleetColumns = [
  col.accessor('phase', {
    header: 'Phase',
    cell: info => <PhaseBadge phase={info.getValue()} />,
    size: 130,
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
        <span className="font-medium text-foreground">
          {resolvedName}
        </span>
      )
    },
  }),
  col.display({
    id: 'duration',
    header: 'Duration',
    cell: ({ row }) => {
      const { startTime, completionTime, phase } = row.original
      if (!startTime) return <span className="text-muted-foreground">—</span>
      const isActive = RUNNING_PHASES.has(phase)
      const endTime = isActive ? null
        : (completionTime && new Date(completionTime) > new Date(startTime)) ? completionTime
        : null
      return (
        <span className="text-muted-foreground font-mono text-xs">
          {formatDuration(startTime, endTime)}
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
    cell: ({ row }) => {
      const { phase, completionTime, updatedAt } = row.original
      if (RUNNING_PHASES.has(phase)) {
        return (
          <span className="text-xs font-medium text-green-600 dark:text-green-400">
            Active now
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
    header: '',
    cell: ({ row }) => <ChatColumnButton sessionId={row.original.id} />,
    size: 48,
  }),
]
