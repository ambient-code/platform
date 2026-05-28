import { createColumnHelper } from '@tanstack/react-table'
import type { DomainSession } from '@/domain/types'
import { formatRelativeTime, formatDuration } from '@/lib/format-timestamp'
import { PhaseBadge } from './phase-badge'

const COST_ANNOTATION = 'ambient-code.io/cost/estimate'
const col = createColumnHelper<DomainSession>()

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
  col.accessor('agentName', {
    header: 'Agent',
    cell: info => {
      const name = info.getValue()
      const agentId = info.row.original.agentId
      return (
        <span className="text-muted-foreground">
          {name ?? agentId ?? '—'}
        </span>
      )
    },
  }),
  col.display({
    id: 'duration',
    header: 'Duration',
    cell: ({ row }) => {
      const { startTime, completionTime } = row.original
      if (!startTime) return <span className="text-muted-foreground">—</span>
      return (
        <span className="text-muted-foreground font-mono text-xs">
          {formatDuration(startTime, completionTime)}
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
  col.accessor('updatedAt', {
    header: 'Last Activity',
    cell: info => (
      <span className="text-muted-foreground text-xs">
        {formatRelativeTime(info.getValue())}
      </span>
    ),
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
]
