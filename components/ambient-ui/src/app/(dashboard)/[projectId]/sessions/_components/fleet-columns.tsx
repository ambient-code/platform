import Link from 'next/link'
import { createColumnHelper } from '@tanstack/react-table'
import type { SortingFn } from '@tanstack/react-table'
import {
  MessageSquare,
  Ticket,
  GitPullRequest,
  ExternalLink,
  Clock,
  CalendarClock,
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Checkbox } from '@/components/ui/checkbox'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import type { DomainSession, SessionPhase } from '@/domain/types'
import { formatRelativeTime, formatAbsoluteTime, formatPreciseDuration } from '@/lib/format-timestamp'
import { useChatSidebar } from '@/components/chat-sidebar-context'
import { PhaseBadge } from './phase-badge'

const COST_ANNOTATION = 'ambient-code.io/cost/estimate'

/** Annotation keys for work item integrations, in priority order */
const WORK_ITEM_ANNOTATIONS = [
  { key: 'ambient-code.io/jira/issue', label: 'Jira', Icon: Ticket },
  { key: 'ambient-code.io/github/pr', label: 'PR', Icon: GitPullRequest },
  { key: 'ambient-code.io/gitlab/mr', label: 'MR', Icon: GitPullRequest },
  { key: 'ambient-code.io/gerrit/change', label: 'Gerrit', Icon: ExternalLink },
] as const

const REVIEW_STATUS_ANNOTATION = 'ambient-code.io/review/status'

type ReviewStatus = 'needs-review' | 'approved' | 'changes-requested'

const REVIEW_STATUS_CONFIG: Record<ReviewStatus, { label: string; className: string }> = {
  'needs-review': {
    label: 'Needs Review',
    className: 'bg-status-warning text-status-warning-foreground border-status-warning-border',
  },
  approved: {
    label: 'Approved',
    className: 'bg-status-success text-status-success-foreground border-status-success-border',
  },
  'changes-requested': {
    label: 'Changes Requested',
    className: 'bg-status-error text-status-error-foreground border-status-error-border',
  },
}

function isReviewStatus(value: string): value is ReviewStatus {
  return value in REVIEW_STATUS_CONFIG
}

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

export type FleetTableMeta = {
  agentNames?: Map<string, string>
  useAbsoluteTime?: boolean
  onToggleTimeFormat?: () => void
}

function ChatColumnButton({ sessionId, sessionName, phase }: { sessionId: string; sessionName: string; phase: SessionPhase }) {
  const { openSidebar, activeSessionId } = useChatSidebar()
  const isActive = activeSessionId === sessionId
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
            openSidebar(sessionId, sessionName)
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
  col.display({
    id: 'select',
    header: ({ table }) => (
      <Checkbox
        checked={
          table.getIsAllPageRowsSelected() ||
          (table.getIsSomePageRowsSelected() && 'indeterminate')
        }
        onCheckedChange={(value) => table.toggleAllPageRowsSelected(!!value)}
        aria-label="Select all"
        onClick={(e) => e.stopPropagation()}
      />
    ),
    cell: ({ row }) => (
      <Checkbox
        checked={row.getIsSelected()}
        onCheckedChange={(value) => row.toggleSelected(!!value)}
        aria-label="Select row"
        onClick={(e) => e.stopPropagation()}
      />
    ),
    size: 40,
    enableSorting: false,
  }),
  col.accessor('phase', {
    header: 'Phase',
    cell: info => <PhaseBadge phase={info.getValue()} />,
    size: 130,
    enableSorting: true,
    sortingFn: phaseSortingFn,
  }),
  col.accessor('name', {
    header: 'Name',
    cell: info => {
      const isTest = info.row.original.annotations['ambient-code.io/ui/test-session'] === 'true'
      return (
        <span className="font-medium">
          {info.getValue()}
          {isTest && (
            <span className="ml-1.5 inline-flex items-center rounded border border-border bg-muted px-1 py-0.5 text-[10px] text-muted-foreground align-middle">
              test
            </span>
          )}
        </span>
      )
    },
  }),
  col.display({
    id: 'workItem',
    header: 'Work Item',
    cell: ({ row }) => {
      const annotations = row.original.annotations
      for (const { key, label, Icon } of WORK_ITEM_ANNOTATIONS) {
        const value = annotations[key]
        if (value) {
          return (
            <span className="inline-flex items-center gap-1 rounded-md border border-border bg-muted px-1.5 py-0.5 text-xs text-muted-foreground">
              <Icon className="size-3 shrink-0" />
              <span className="truncate max-w-[120px]">{value}</span>
            </span>
          )
        }
      }
      return <span className="text-muted-foreground">—</span>
    },
    enableSorting: false,
    size: 160,
  }),
  col.accessor('agentId', {
    header: 'Agent',
    cell: info => {
      const agentId = info.getValue()
      const projectId = info.row.original.projectId
      const annotationName = info.row.original.agentName
      const agentNames = (info.table.options.meta as FleetTableMeta | undefined)?.agentNames
      const resolvedName = annotationName ?? (agentId ? agentNames?.get(agentId) : null) ?? null
      if (!resolvedName) return <span className="text-muted-foreground">—</span>
      if (!agentId || !projectId) return <span className="text-sm text-foreground">{resolvedName}</span>
      return (
        <Link
          href={`/${projectId}/agents/${agentId}`}
          onClick={e => e.stopPropagation()}
          className="text-sm text-link underline-offset-4 hover:underline"
        >
          {resolvedName}
        </Link>
      )
    },
  }),
  col.display({
    id: 'review',
    header: 'Review',
    cell: ({ row }) => {
      const raw = row.original.annotations[REVIEW_STATUS_ANNOTATION]
      if (!raw || !isReviewStatus(raw)) {
        return <span className="text-muted-foreground">—</span>
      }
      const config = REVIEW_STATUS_CONFIG[raw]
      return (
        <Badge variant="outline" className={config.className}>
          {config.label}
        </Badge>
      )
    },
    enableSorting: false,
    size: 140,
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
    header: ({ table }) => {
      const meta = table.options.meta as FleetTableMeta | undefined
      const isAbsolute = meta?.useAbsoluteTime ?? false
      const toggle = meta?.onToggleTimeFormat
      return (
        <div className="flex items-center gap-1">
          <span>Last Activity</span>
          {toggle && (
            <Tooltip>
              <TooltipTrigger asChild>
                <button
                  type="button"
                  className="inline-flex items-center justify-center rounded-sm p-0.5 text-muted-foreground hover:text-foreground transition-colors"
                  onClick={(e) => {
                    e.stopPropagation()
                    toggle()
                  }}
                  aria-label={isAbsolute ? 'Switch to relative time' : 'Switch to absolute time'}
                >
                  {isAbsolute ? (
                    <Clock className="size-3.5" />
                  ) : (
                    <CalendarClock className="size-3.5" />
                  )}
                </button>
              </TooltipTrigger>
              <TooltipContent>
                {isAbsolute ? 'Show relative time' : 'Show absolute time'}
              </TooltipContent>
            </Tooltip>
          )}
        </div>
      )
    },
    enableSorting: true,
    sortingFn: (rowA, rowB) => {
      const getTime = (row: typeof rowA) => {
        const { phase, completionTime, updatedAt } = row.original
        if (RUNNING_PHASES.has(phase)) return Date.now()
        return new Date(completionTime ?? updatedAt).getTime()
      }
      return getTime(rowA) - getTime(rowB)
    },
    cell: ({ row, table }) => {
      const { phase, completionTime, updatedAt } = row.original
      const meta = table.options.meta as FleetTableMeta | undefined
      const useAbsolute = meta?.useAbsoluteTime ?? false

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
          {useAbsolute ? formatAbsoluteTime(activityTime) : formatRelativeTime(activityTime)}
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
    header: '',
    cell: ({ row }) => <ChatColumnButton sessionId={row.original.id} sessionName={row.original.name} phase={row.original.phase} />,
    size: 48,
    enableSorting: false,
  }),
]
