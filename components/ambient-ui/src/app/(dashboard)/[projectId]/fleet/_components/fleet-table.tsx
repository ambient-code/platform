'use client'

import { useRouter, useParams } from 'next/navigation'
import {
  useReactTable,
  getCoreRowModel,
  getFilteredRowModel,
  flexRender,
} from '@tanstack/react-table'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { TooltipProvider } from '@/components/ui/tooltip'
import type { DomainSession } from '@/domain/types'
import { fleetColumns } from './fleet-columns'

export function FleetTable({
  sessions,
  searchFilter,
}: {
  sessions: DomainSession[]
  searchFilter: string
}) {
  const router = useRouter()
  const { projectId } = useParams<{ projectId: string }>()

  const table = useReactTable({
    data: sessions,
    columns: fleetColumns,
    getCoreRowModel: getCoreRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
    globalFilterFn: 'includesString',
    state: { globalFilter: searchFilter },
  })

  return (
    <TooltipProvider>
    <div className="rounded-md border">
      <Table>
        <TableHeader>
          {table.getHeaderGroups().map(headerGroup => (
            <TableRow key={headerGroup.id}>
              {headerGroup.headers.map(header => (
                <TableHead key={header.id}>
                  {header.isPlaceholder
                    ? null
                    : flexRender(header.column.columnDef.header, header.getContext())}
                </TableHead>
              ))}
            </TableRow>
          ))}
        </TableHeader>
        <TableBody>
          {table.getRowModel().rows.length ? (
            table.getRowModel().rows.map(row => (
              <TableRow
                key={row.id}
                className="cursor-pointer"
                onClick={() => router.push(`/${projectId}/fleet/${row.original.id}`)}
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
