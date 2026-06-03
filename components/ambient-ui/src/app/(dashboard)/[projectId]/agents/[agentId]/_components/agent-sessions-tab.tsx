'use client'

import { useState, useMemo } from 'react'
import Link from 'next/link'
import {
  useReactTable,
  getCoreRowModel,
  getSortedRowModel,
  createColumnHelper,
  flexRender,
} from '@tanstack/react-table'
import type { SortingState } from '@tanstack/react-table'
import { ChevronUp, ChevronDown, Monitor } from 'lucide-react'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Skeleton } from '@/components/ui/skeleton'
import { EmptyState } from '@/components/empty-state'
import { PhaseBadge } from '../../../sessions/_components/phase-badge'
import { formatRelativeTime, formatDuration } from '@/lib/format-timestamp'
import { useSessions } from '@/queries/use-sessions'
import type { DomainSession } from '@/domain/types'

const COST_KEY = 'ambient-code.io/cost/estimate'

const col = createColumnHelper<DomainSession>()

function buildColumns(projectId: string) {
  return [
    col.accessor('phase', {
      header: 'Phase',
      cell: (info) => <PhaseBadge phase={info.getValue()} />,
    }),
    col.accessor('name', {
      header: 'Name',
      cell: (info) => (
        <Link
          href={`/${projectId}/sessions/${info.row.original.id}`}
          className="text-sm font-medium text-link underline-offset-4 hover:underline"
        >
          {info.getValue()}
        </Link>
      ),
    }),
    col.accessor(
      (row) => {
        if (!row.startTime) return -1
        const end = row.completionTime ? new Date(row.completionTime).getTime() : Date.now()
        return end - new Date(row.startTime).getTime()
      },
      {
        id: 'duration',
        header: 'Duration',
        cell: ({ row }) => (
          <span className="text-sm text-muted-foreground">
            {row.original.startTime
              ? formatDuration(row.original.startTime, row.original.completionTime)
              : '—'}
          </span>
        ),
      },
    ),
    col.accessor(
      (row) => {
        const raw = row.annotations[COST_KEY]
        if (!raw) return -1
        return parseFloat(raw.replace(/[^0-9.]/g, '')) || 0
      },
      {
        id: 'cost',
        header: 'Cost',
        cell: ({ row }) => {
          const cost = row.original.annotations[COST_KEY]
          return (
            <span className="text-sm text-muted-foreground">
              {cost ?? '—'}
            </span>
          )
        },
      },
    ),
    col.accessor('createdAt', {
      header: 'Created',
      cell: (info) => (
        <span className="text-xs text-muted-foreground">
          {info.getValue() ? formatRelativeTime(info.getValue()) : '—'}
        </span>
      ),
    }),
  ]
}

export function AgentSessionsTab({
  agentId,
  projectId,
}: {
  agentId: string
  projectId: string
}) {
  const { data, isLoading, error } = useSessions(projectId, { size: 100 })
  const [sorting, setSorting] = useState<SortingState>([
    { id: 'createdAt', desc: true },
  ])

  const agentSessions = useMemo(
    () => (data?.items ?? []).filter((session) => session.agentId === agentId),
    [data?.items, agentId],
  )

  const columns = useMemo(() => buildColumns(projectId), [projectId])

  const table = useReactTable({
    data: agentSessions,
    columns,
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
    state: { sorting },
    onSortingChange: setSorting,
  })

  if (error) {
    return (
      <p className="text-sm text-destructive pt-4">
        Failed to load sessions: {error.message}
      </p>
    )
  }

  if (isLoading) {
    return (
      <div className="space-y-3 pt-4">
        <Skeleton className="h-[300px] w-full" />
      </div>
    )
  }

  if (agentSessions.length === 0) {
    return (
      <div className="pt-4">
        <EmptyState
          icon={Monitor}
          title="No sessions"
          description="This agent has no sessions yet. Run a test session to get started."
        />
      </div>
    )
  }

  return (
    <div className="rounded-md border mt-4">
      <Table>
        <TableHeader>
          {table.getHeaderGroups().map((headerGroup) => (
            <TableRow key={headerGroup.id}>
              {headerGroup.headers.map((header) => {
                const canSort = header.column.getCanSort()
                const sorted = header.column.getIsSorted()
                return (
                  <TableHead
                    key={header.id}
                    className={canSort ? 'cursor-pointer select-none' : undefined}
                    onClick={canSort ? header.column.getToggleSortingHandler() : undefined}
                  >
                    <div className="flex items-center gap-1">
                      {header.isPlaceholder
                        ? null
                        : flexRender(header.column.columnDef.header, header.getContext())}
                      {canSort && sorted === 'asc' && <ChevronUp className="size-3.5 text-foreground" />}
                      {canSort && sorted === 'desc' && <ChevronDown className="size-3.5 text-foreground" />}
                      {canSort && !sorted && <ChevronDown className="size-3.5 text-muted-foreground/40" />}
                    </div>
                  </TableHead>
                )
              })}
            </TableRow>
          ))}
        </TableHeader>
        <TableBody>
          {table.getRowModel().rows.map((row) => (
            <TableRow key={row.id}>
              {row.getVisibleCells().map((cell) => (
                <TableCell key={cell.id}>
                  {flexRender(cell.column.columnDef.cell, cell.getContext())}
                </TableCell>
              ))}
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  )
}
