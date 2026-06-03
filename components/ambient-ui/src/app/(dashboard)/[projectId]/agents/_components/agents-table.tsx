'use client'

import { useState, useRef, useCallback } from 'react'
import { useRouter, useParams } from 'next/navigation'
import {
  useReactTable,
  getCoreRowModel,
  getFilteredRowModel,
  getSortedRowModel,
  createColumnHelper,
  flexRender,
} from '@tanstack/react-table'
import type { SortingState } from '@tanstack/react-table'
import { ChevronUp, ChevronDown } from 'lucide-react'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import type { DomainAgent } from '@/domain/types'
import { formatRelativeTime } from '@/lib/format-timestamp'
import { useTableKeyboardNav } from '@/hooks/use-table-keyboard-nav'
import { cn } from '@/lib/utils'
import { LifecycleBadge, getAgentLifecycle } from './lifecycle-badge'

const col = createColumnHelper<DomainAgent>()

const agentColumns = [
  col.accessor((row) => row.displayName ?? row.name, {
    id: 'name',
    header: 'Name',
    cell: info => (
      <div>
        <span className="font-medium">{info.getValue()}</span>
        {info.row.original.displayName && (
          <span className="ml-2 text-xs text-muted-foreground">
            {info.row.original.name}
          </span>
        )}
      </div>
    ),
  }),
  col.display({
    id: 'source',
    header: 'Source',
    cell: ({ row }) => {
      const lifecycle = getAgentLifecycle(row.original.annotations)
      return <LifecycleBadge lifecycle={lifecycle} />
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
  col.accessor('ownerUserId', {
    header: 'Owner',
    cell: info => (
      <span className="text-sm text-muted-foreground">
        {info.getValue() ?? '—'}
      </span>
    ),
  }),
  col.accessor('currentSessionId', {
    header: 'Current Session',
    cell: info => {
      const sessionId = info.getValue()
      if (!sessionId) return <span className="text-muted-foreground">{'—'}</span>
      return (
        <span className="text-xs font-mono text-foreground truncate max-w-[120px] inline-block">
          {sessionId}
        </span>
      )
    },
  }),
  col.display({
    id: 'lastActive',
    header: 'Last Active',
    enableSorting: true,
    sortingFn: (rowA, rowB) => {
      return new Date(rowA.original.updatedAt).getTime() - new Date(rowB.original.updatedAt).getTime()
    },
    cell: ({ row }) => (
      <span className="text-muted-foreground text-xs">
        {row.original.updatedAt ? formatRelativeTime(row.original.updatedAt) : '—'}
      </span>
    ),
  }),
]

export function AgentsTable({
  agents,
  searchFilter,
}: {
  agents: DomainAgent[]
  searchFilter: string
}) {
  const router = useRouter()
  const { projectId } = useParams<{ projectId: string }>()
  const containerRef = useRef<HTMLDivElement>(null)
  const [sorting, setSorting] = useState<SortingState>([
    { id: 'lastActive', desc: true },
  ])

  const table = useReactTable({
    data: agents,
    columns: agentColumns,
    getCoreRowModel: getCoreRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
    getSortedRowModel: getSortedRowModel(),
    globalFilterFn: 'includesString',
    state: {
      globalFilter: searchFilter,
      sorting,
    },
    onSortingChange: setSorting,
  })

  const visibleRows = table.getRowModel().rows

  const navigateToAgent = useCallback(
    (agent: DomainAgent) => {
      router.push(`/${projectId}/agents/${agent.id}`)
    },
    [router, projectId],
  )

  const handleKeyboardSelect = useCallback(
    (index: number) => {
      const row = visibleRows[index]
      if (row) {
        navigateToAgent(row.original)
      }
    },
    [visibleRows, navigateToAgent],
  )

  const { selectedIndex } = useTableKeyboardNav({
    rowCount: visibleRows.length,
    onSelect: handleKeyboardSelect,
    containerRef,
  })

  return (
    <div ref={containerRef} tabIndex={-1} className="rounded-md border outline-none">
      <Table>
        <TableHeader>
          {table.getHeaderGroups().map(headerGroup => (
            <TableRow key={headerGroup.id}>
              {headerGroup.headers.map(header => {
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
                      {canSort && sorted === 'asc' && (
                        <ChevronUp className="size-3.5 text-foreground" />
                      )}
                      {canSort && sorted === 'desc' && (
                        <ChevronDown className="size-3.5 text-foreground" />
                      )}
                      {canSort && !sorted && (
                        <ChevronDown className="size-3.5 text-muted-foreground/40" />
                      )}
                    </div>
                  </TableHead>
                )
              })}
            </TableRow>
          ))}
        </TableHeader>
        <TableBody>
          {visibleRows.length ? (
            visibleRows.map((row, rowIndex) => (
              <TableRow
                key={row.id}
                className={cn(
                  'cursor-pointer group',
                  rowIndex === selectedIndex && 'bg-muted ring-2 ring-ring ring-inset',
                )}
                tabIndex={0}
                data-state={rowIndex === selectedIndex ? 'selected' : undefined}
                onClick={() => navigateToAgent(row.original)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter') {
                    navigateToAgent(row.original)
                  }
                }}
              >
                {row.getVisibleCells().map(cell => (
                  <TableCell key={cell.id}>
                    {flexRender(cell.column.columnDef.cell, cell.getContext())}
                  </TableCell>
                ))}
              </TableRow>
            ))
          ) : (
            <TableRow>
              <TableCell colSpan={agentColumns.length} className="h-24 text-center text-muted-foreground">
                No agents match your filter.
              </TableCell>
            </TableRow>
          )}
        </TableBody>
      </Table>
    </div>
  )
}
