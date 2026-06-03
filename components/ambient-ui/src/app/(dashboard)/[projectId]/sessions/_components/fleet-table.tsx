'use client'

import { useState, useEffect, useRef, useCallback } from 'react'
import { useRouter, useParams } from 'next/navigation'
import {
  useReactTable,
  getCoreRowModel,
  getFilteredRowModel,
  getSortedRowModel,
  flexRender,
} from '@tanstack/react-table'
import type { SortingState, ColumnFiltersState, RowSelectionState } from '@tanstack/react-table'
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
import { useTableKeyboardNav } from '@/hooks/use-table-keyboard-nav'
import { cn } from '@/lib/utils'
import { fleetColumns } from './fleet-columns'
import type { FleetTableMeta } from './fleet-columns'
import { BulkActionBar } from './bulk-action-bar'

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
  const containerRef = useRef<HTMLDivElement>(null)

  const [sorting, setSorting] = useState<SortingState>([
    { id: 'phase', desc: false },
    { id: 'lastActivity', desc: true },
  ])

  const [columnFilters, setColumnFilters] = useState<ColumnFiltersState>([])
  const [rowSelection, setRowSelection] = useState<RowSelectionState>({})
  const [useAbsoluteTime, setUseAbsoluteTime] = useState(false)

  const handleToggleTimeFormat = useCallback(() => {
    setUseAbsoluteTime(prev => !prev)
  }, [])

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

  const tableMeta: FleetTableMeta = {
    agentNames,
    useAbsoluteTime,
    onToggleTimeFormat: handleToggleTimeFormat,
  }

  const table = useReactTable({
    data: sessions,
    columns: fleetColumns,
    getCoreRowModel: getCoreRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
    getSortedRowModel: getSortedRowModel(),
    globalFilterFn: 'includesString',
    enableRowSelection: true,
    state: {
      globalFilter: searchFilter,
      sorting,
      columnFilters,
      rowSelection,
    },
    onSortingChange: setSorting,
    onColumnFiltersChange: setColumnFilters,
    onRowSelectionChange: setRowSelection,
    meta: tableMeta,
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

  // Clear selection when data changes (e.g., after bulk stop/delete)
  useEffect(() => {
    setRowSelection({})
  }, [sessions.length])

  const visibleRows = table.getRowModel().rows
  const handleKeyboardSelect = useCallback(
    (index: number) => {
      const row = visibleRows[index]
      if (row) {
        router.push(`/${projectId}/sessions/${row.original.id}`)
      }
    },
    [visibleRows, router, projectId],
  )

  const { selectedIndex } = useTableKeyboardNav({
    rowCount: visibleRows.length,
    onSelect: handleKeyboardSelect,
    containerRef,
  })

  const selectedRows = table.getSelectedRowModel().rows
  const selectedSessions = selectedRows.map(row => row.original)

  const handleClearSelection = useCallback(() => {
    setRowSelection({})
  }, [])

  return (
    <TooltipProvider delayDuration={300}>
      {selectedSessions.length > 0 && (
        <BulkActionBar
          selectedSessions={selectedSessions}
          onClearSelection={handleClearSelection}
        />
      )}
    <div ref={containerRef} tabIndex={-1} className="rounded-md border outline-none">
      <Table>
        <TableHeader>
          {table.getHeaderGroups().map(headerGroup => (
            <TableRow key={headerGroup.id}>
              {headerGroup.headers.map(header => {
                const canSort = header.column.getCanSort()
                const sorted = header.column.getIsSorted()
                const isChat = header.column.id === 'chat'
                const isSelect = header.column.id === 'select'

                return (
                  <TableHead
                    key={header.id}
                    {...(isChat ? { 'data-sticky': 'right' } : {})}
                    className={canSort && !isSelect ? 'cursor-pointer select-none' : undefined}
                    onClick={canSort && !isSelect ? header.column.getToggleSortingHandler() : undefined}
                    style={header.column.columnDef.size ? { width: header.column.columnDef.size } : undefined}
                  >
                    <div className="flex items-center gap-1">
                      {header.isPlaceholder
                        ? null
                        : flexRender(header.column.columnDef.header, header.getContext())}
                      {canSort && !isSelect && sorted === 'asc' && (
                        <ChevronUp className="size-3.5 text-foreground" />
                      )}
                      {canSort && !isSelect && sorted === 'desc' && (
                        <ChevronDown className="size-3.5 text-foreground" />
                      )}
                      {canSort && !isSelect && !sorted && (
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
                data-state={
                  row.getIsSelected()
                    ? 'selected'
                    : rowIndex === selectedIndex
                      ? 'selected'
                      : undefined
                }
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
