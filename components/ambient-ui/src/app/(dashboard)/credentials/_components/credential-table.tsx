'use client'

import { useState, useMemo } from 'react'
import {
  useReactTable,
  getCoreRowModel,
  getSortedRowModel,
  getFilteredRowModel,
  getExpandedRowModel,
  getGroupedRowModel,
  createColumnHelper,
  flexRender,
} from '@tanstack/react-table'
import type { SortingState, ExpandedState } from '@tanstack/react-table'
import { ChevronUp, ChevronDown, ChevronRight } from 'lucide-react'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import type { DomainCredential, DomainRoleBinding } from '@/domain/types'
import { getCategoryForProvider, getProviderMeta } from '@/domain/credential-providers'
import { formatRelativeTime } from '@/lib/format-timestamp'
import { CredentialManageSheet } from './credential-manage-sheet'

type CredentialRow = DomainCredential & {
  category: string
  bindingCount: number
}

const col = createColumnHelper<CredentialRow>()

function ProviderBadge({ provider }: { provider: string }) {
  const meta = getProviderMeta(provider)
  return (
    <Badge variant="outline" className="font-normal">
      {meta?.label ?? provider}
    </Badge>
  )
}

const credentialColumns = [
  col.accessor('category', {
    header: 'Category',
    enableGrouping: true,
    cell: info => info.getValue(),
  }),
  col.accessor('name', {
    header: 'Name',
    cell: info => (
      <span className="font-medium">{info.getValue()}</span>
    ),
  }),
  col.accessor('provider', {
    header: 'Provider',
    cell: info => <ProviderBadge provider={info.getValue()} />,
  }),
  col.accessor('description', {
    header: 'Description',
    cell: info => {
      const value = info.getValue()
      if (!value) return <span className="text-muted-foreground">--</span>
      return (
        <span className="text-sm text-muted-foreground truncate max-w-[200px] inline-block">
          {value}
        </span>
      )
    },
  }),
  col.accessor('bindingCount', {
    id: 'bindings',
    header: 'Bindings',
    cell: ({ row, getValue }) => {
      if (row.getIsGrouped()) return null
      const count = getValue()
      if (count === 0) {
        return <span className="text-muted-foreground">0</span>
      }
      return <span>{count}</span>
    },
  }),
  col.accessor('createdAt', {
    header: 'Created',
    cell: info => (
      <span className="text-muted-foreground text-xs">
        {formatRelativeTime(info.getValue())}
      </span>
    ),
  }),
]

export function CredentialTable({
  credentials,
  bindings,
}: {
  credentials: DomainCredential[]
  bindings: DomainRoleBinding[]
}) {
  const [search, setSearch] = useState('')
  const [sorting, setSorting] = useState<SortingState>([])
  const [expanded, setExpanded] = useState<ExpandedState>(true)
  const [selectedCredential, setSelectedCredential] = useState<DomainCredential | null>(null)

  const rows: CredentialRow[] = useMemo(
    () =>
      credentials.map((c) => ({
        ...c,
        category: getCategoryForProvider(c.provider) ?? 'Other',
        bindingCount: bindings.filter((b) => b.credentialId === c.id).length,
      })),
    [credentials, bindings],
  )

  const table = useReactTable({
    data: rows,
    columns: credentialColumns,
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
    getGroupedRowModel: getGroupedRowModel(),
    getExpandedRowModel: getExpandedRowModel(),
    globalFilterFn: 'includesString',
    state: {
      globalFilter: search,
      sorting,
      expanded,
      grouping: ['category'],
    },
    onSortingChange: setSorting,
    onExpandedChange: setExpanded,
  })

  return (
    <>
      <div className="flex items-center gap-2 mb-3">
        <Input
          placeholder="Filter credentials..."
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          className="max-w-xs"
        />
      </div>

      <div className="rounded-md border">
        <Table>
          <TableHeader>
            {table.getHeaderGroups().map((headerGroup) => (
              <TableRow key={headerGroup.id}>
                {headerGroup.headers.map((header) => {
                  if (header.column.id === 'category') return null
                  const canSort = header.column.getCanSort()
                  const sorted = header.column.getIsSorted()

                  return (
                    <TableHead
                      key={header.id}
                      colSpan={header.colSpan}
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
              table.getRowModel().rows.map((row) => {
                if (row.getIsGrouped()) {
                  const isExpanded = row.getIsExpanded()
                  return (
                    <TableRow
                      key={row.id}
                      className="cursor-pointer bg-muted/50 hover:bg-muted"
                      onClick={() => row.toggleExpanded()}
                    >
                      <TableCell colSpan={credentialColumns.length - 1}>
                        <div className="flex items-center gap-2">
                          <ChevronRight
                            className={`size-4 transition-transform ${isExpanded ? 'rotate-90' : ''}`}
                          />
                          <span className="text-sm font-medium">
                            {row.groupingValue as string}
                          </span>
                          <Badge variant="secondary" className="text-xs">
                            {row.subRows.length}
                          </Badge>
                        </div>
                      </TableCell>
                    </TableRow>
                  )
                }

                return (
                  <TableRow
                    key={row.id}
                    className="cursor-pointer"
                    onClick={() => setSelectedCredential(row.original)}
                  >
                    {row.getVisibleCells().map((cell) => {
                      if (cell.column.id === 'category') return null
                      return (
                        <TableCell key={cell.id}>
                          {cell.getIsAggregated()
                            ? null
                            : flexRender(cell.column.columnDef.cell, cell.getContext())}
                        </TableCell>
                      )
                    })}
                  </TableRow>
                )
              })
            ) : (
              <TableRow>
                <TableCell
                  colSpan={credentialColumns.length - 1}
                  className="h-24 text-center text-muted-foreground"
                >
                  No credentials match your filter.
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </div>

      <CredentialManageSheet
        credential={selectedCredential}
        open={selectedCredential !== null}
        onOpenChange={(open) => {
          if (!open) setSelectedCredential(null)
        }}
      />
    </>
  )
}
