'use client'

import { useState } from 'react'
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
  onSelectAgent,
}: {
  agents: DomainAgent[]
  searchFilter: string
  onSelectAgent: (agent: DomainAgent) => void
}) {
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

  return (
    <div className="rounded-md border">
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
          {table.getRowModel().rows.length ? (
            table.getRowModel().rows.map(row => (
              <TableRow
                key={row.id}
                className="cursor-pointer group"
                tabIndex={0}
                onClick={() => onSelectAgent(row.original)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter') {
                    onSelectAgent(row.original)
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
