"use client";

import * as React from "react";
import {
  type ColumnDef,
  type ColumnFiltersState,
  type SortingState,
  type VisibilityState,
  flexRender,
  getCoreRowModel,
  getFilteredRowModel,
  getPaginationRowModel,
  getSortedRowModel,
  useReactTable,
} from "@tanstack/react-table";
import { ArrowUpDown, ChevronDown, Settings2 } from "lucide-react";

import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Input } from "@/components/ui/input";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

// Re-export ColumnDef for consumers to define their columns
export type { ColumnDef } from "@tanstack/react-table";

/**
 * Helper to create a sortable header for a column.
 * Usage in column definitions:
 *
 * ```tsx
 * {
 *   accessorKey: "name",
 *   header: sortableHeader("Name"),
 * }
 * ```
 */
export function sortableHeader<TData>(label: string) {
  return function SortableHeader({
    column,
  }: {
    column: { toggleSorting: (desc?: boolean) => void; getIsSorted: () => false | "asc" | "desc" };
  }) {
    return (
      <Button
        variant="ghost"
        onClick={() => {
          const sorted = column.getIsSorted();
          column.toggleSorting(sorted === "asc");
        }}
        className="-ml-4"
      >
        {label}
        <ArrowUpDown className="ml-2 h-4 w-4" />
      </Button>
    );
  } as React.FC<{ column: Parameters<Exclude<ColumnDef<TData, unknown>["header"], string | undefined>>[0]["column"] }>;
}

interface DataTableProps<TData, TValue> {
  /** Column definitions following TanStack Table ColumnDef pattern */
  columns: ColumnDef<TData, TValue>[];
  /** Data array to render in the table */
  data: TData[];
  /** Show a text filter input above the table */
  searchable?: boolean;
  /** Placeholder text for the search input */
  searchPlaceholder?: string;
  /**
   * The column ID to filter on when using the search input.
   * If not provided, a global text filter is applied across all columns.
   */
  searchColumnId?: string;
  /** Enable pagination (defaults to false, showing all rows) */
  paginated?: boolean;
  /** Number of rows per page when paginated (default: 10) */
  pageSize?: number;
  /** Message displayed when the table has no data */
  emptyMessage?: string;
  /** Show a column visibility toggle dropdown */
  columnVisibility?: boolean;
}

export function DataTable<TData, TValue>({
  columns,
  data,
  searchable = false,
  searchPlaceholder = "Search...",
  searchColumnId,
  paginated = false,
  pageSize = 10,
  emptyMessage = "No results.",
  columnVisibility: showColumnVisibility = false,
}: DataTableProps<TData, TValue>) {
  const [sorting, setSorting] = React.useState<SortingState>([]);
  const [columnFilters, setColumnFilters] = React.useState<ColumnFiltersState>([]);
  const [columnVisibilityState, setColumnVisibilityState] = React.useState<VisibilityState>({});
  const [rowSelection, setRowSelection] = React.useState({});
  const [globalFilter, setGlobalFilter] = React.useState("");

  const table = useReactTable({
    data,
    columns,
    onSortingChange: setSorting,
    onColumnFiltersChange: setColumnFilters,
    onGlobalFilterChange: setGlobalFilter,
    getCoreRowModel: getCoreRowModel(),
    ...(paginated ? { getPaginationRowModel: getPaginationRowModel() } : {}),
    getSortedRowModel: getSortedRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
    onColumnVisibilityChange: setColumnVisibilityState,
    onRowSelectionChange: setRowSelection,
    globalFilterFn: "includesString",
    state: {
      sorting,
      columnFilters,
      columnVisibility: columnVisibilityState,
      rowSelection,
      globalFilter,
    },
    initialState: {
      pagination: {
        pageSize,
      },
    },
  });

  const handleSearchChange = (value: string) => {
    if (searchColumnId) {
      table.getColumn(searchColumnId)?.setFilterValue(value);
    } else {
      setGlobalFilter(value);
    }
  };

  const searchValue = searchColumnId
    ? (table.getColumn(searchColumnId)?.getFilterValue() as string) ?? ""
    : globalFilter;

  return (
    <div className="space-y-4">
      {/* Toolbar: search + column visibility */}
      {(searchable || showColumnVisibility) && (
        <div className="flex items-center gap-2">
          {searchable && (
            <Input
              placeholder={searchPlaceholder}
              value={searchValue}
              onChange={(e) => handleSearchChange(e.target.value)}
              className="max-w-sm"
            />
          )}
          {showColumnVisibility && (
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="outline" className="ml-auto">
                  <Settings2 className="mr-2 h-4 w-4" />
                  Columns
                  <ChevronDown className="ml-2 h-4 w-4" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuLabel>Toggle columns</DropdownMenuLabel>
                <DropdownMenuSeparator />
                {table
                  .getAllColumns()
                  .filter((column) => column.getCanHide())
                  .map((column) => (
                    <DropdownMenuCheckboxItem
                      key={column.id}
                      className="capitalize"
                      checked={column.getIsVisible()}
                      onCheckedChange={(value) => column.toggleVisibility(!!value)}
                    >
                      {column.id}
                    </DropdownMenuCheckboxItem>
                  ))}
              </DropdownMenuContent>
            </DropdownMenu>
          )}
        </div>
      )}

      {/* Table */}
      <div className="rounded-md border">
        <Table>
          <TableHeader>
            {table.getHeaderGroups().map((headerGroup) => (
              <TableRow key={headerGroup.id}>
                {headerGroup.headers.map((header) => (
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
              table.getRowModel().rows.map((row) => (
                <TableRow key={row.id} data-state={row.getIsSelected() && "selected"}>
                  {row.getVisibleCells().map((cell) => (
                    <TableCell key={cell.id}>
                      {flexRender(cell.column.columnDef.cell, cell.getContext())}
                    </TableCell>
                  ))}
                </TableRow>
              ))
            ) : (
              <TableRow>
                <TableCell colSpan={columns.length} className="h-24 text-center">
                  {emptyMessage}
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </div>

      {/* Pagination */}
      {paginated && (
        <div className="flex items-center justify-between px-2">
          <div className="text-sm text-muted-foreground">
            {table.getFilteredSelectedRowModel().rows.length > 0 && (
              <span>
                {table.getFilteredSelectedRowModel().rows.length} of{" "}
                {table.getFilteredRowModel().rows.length} row(s) selected.
              </span>
            )}
          </div>
          <div className="flex items-center space-x-2">
            <div className="text-sm text-muted-foreground">
              Page {table.getState().pagination.pageIndex + 1} of {table.getPageCount()}
            </div>
            <Button
              variant="outline"
              size="sm"
              onClick={() => table.previousPage()}
              disabled={!table.getCanPreviousPage()}
            >
              Previous
            </Button>
            <Button
              variant="outline"
              size="sm"
              onClick={() => table.nextPage()}
              disabled={!table.getCanNextPage()}
            >
              Next
            </Button>
          </div>
        </div>
      )}
    </div>
  );
}
