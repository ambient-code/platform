'use client'

import { useState, useEffect } from 'react'
import { useRouter, useParams } from 'next/navigation'
import {
  useReactTable,
  getCoreRowModel,
  getFilteredRowModel,
  getSortedRowModel,
  flexRender,
} from '@tanstack/react-table'
import type { SortingState, ColumnFiltersState } from '@tanstack/react-table'
import { ChevronUp, ChevronDown } from 'lucide-react'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { TooltipProvider } from '@/components/ui/tooltip'
import type { DomainSession, SessionPhase } from '@/domain/types'
import { fleetColumns } from './fleet-columns'

export function FleetTable({
  sessions,
  searchFilter,
  agentNames,
  phaseFilter,
  onFilteredCountChange,
}: {
  sessions: DomainSession[]
  searchFilter: string
  agentNames?: Map<string, string>
  phaseFilter?: SessionPhase | null
  onFilteredCountChange?: (count: number) => void
}) {
  const router = useRouter()
  const { projectId } = useParams<{ projectId: string }>()

  const [sorting, setSorting] = useState<SortingState>([
    { id: 'phase', desc: false },
    { id: 'lastActivity', desc: true },
  ])

  const [columnFilters, setColumnFilters] = useState<ColumnFiltersState>([])

  // Sync phaseFilter prop to column filters
  useEffect(() => {
    setColumnFilters(prev => {
      const without = prev.filter(f => f.id !== 'phase')
      if (phaseFilter) {
        return [...without, { id: 'phase', value: phaseFilter }]
      }
      return without
    })
  }, [phaseFilter])

  const table = useReactTable({
    data: sessions,
    columns: fleetColumns,
    getCoreRowModel: getCoreRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
    getSortedRowModel: getSortedRowModel(),
    globalFilterFn: 'includesString',
    state: {
      globalFilter: searchFilter,
      sorting,
      columnFilters,
    },
    onSortingChange: setSorting,
    onColumnFiltersChange: setColumnFilters,
    meta: { agentNames },
    filterFns: {
      phaseEquals: (row, columnId, filterValue) => {
        return row.getValue(columnId) === filterValue
      },
    },
  })

  // Report filtered count back to parent
  const filteredRowCount = table.getFilteredRowModel().rows.length
  useEffect(() => {
    onFilteredCountChange?.(filteredRowCount)
  }, [filteredRowCount, onFilteredCountChange])

  return (
    <TooltipProvider delayDuration={300}>
    <div className="rounded-md border">
      <Table>
        <TableHeader>
          {table.getHeaderGroups().map(headerGroup => (
            <TableRow key={headerGroup.id}>
              {headerGroup.headers.map(header => {
                const canSort = header.column.getCanSort()
                const sorted = header.column.getIsSorted()
                const isChat = header.column.id === 'chat'

                return (
                  <TableHead
                    key={header.id}
                    {...(isChat ? { 'data-sticky': 'right' } : {})}
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
                onClick={() => router.push(`/${projectId}/sessions/${row.original.id}`)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter') {
                    router.push(`/${projectId}/sessions/${row.original.id}`)
                  }
                }}
              >
                {row.getVisibleCells().map(cell => {
                  const isChat = cell.column.id === 'chat'
                  return (
                    <TableCell
                      key={cell.id}
                      {...(isChat ? { 'data-sticky': 'right' } : {})}
                    >
                      {flexRender(cell.column.columnDef.cell, cell.getContext())}
                    </TableCell>
                  )
                })}
              </TableRow>
            ))
          ) : (
            <TableRow>
              <TableCell colSpan={fleetColumns.length} className="h-24 text-center text-muted-foreground">
                No sessions match your filter.
              </TableCell>
            </TableRow>
          )}
        </TableBody>
      </Table>
    </div>
    </TooltipProvider>
  )
}
