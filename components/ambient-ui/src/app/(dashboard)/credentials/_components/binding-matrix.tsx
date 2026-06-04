'use client'

import {
  useState,
  useMemo,
  useCallback,
  useRef,
  useEffect,
} from 'react'
import { Check, ChevronDown, Loader2, Search, AlertTriangle } from 'lucide-react'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { Separator } from '@/components/ui/separator'
import { useCreateRoleBinding, useDeleteRoleBinding } from '@/queries/use-role-bindings'
import { cn } from '@/lib/utils'
import type { DomainCredential, DomainRoleBinding, DomainProject, DomainAgent } from '@/domain/types'

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

type ProjectGroup = {
  project: DomainProject
  agents: DomainAgent[]
}

type BulkConfirmState = {
  show: boolean
  title: string
  message: string
  count: number
  onConfirm: () => void
}

type BindingMatrixProps = {
  credentials: DomainCredential[]
  projects: DomainProject[]
  agents: DomainAgent[]
  bindings: DomainRoleBinding[]
  roleId: string
}

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

const PAGE_SIZE = 25
const BATCH_CHUNK_SIZE = 10

const INITIAL_BULK_CONFIRM: BulkConfirmState = {
  show: false,
  title: '',
  message: '',
  count: 0,
  onConfirm: () => undefined,
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function cellKey(credentialId: string, targetId: string): string {
  return `${credentialId}:${targetId}`
}

/**
 * A credential is project-bound when there is a RoleBinding with:
 *   credentialId === cred.id && projectId === project.id && !agentId
 */
function findProjectBinding(
  bindings: DomainRoleBinding[],
  credentialId: string,
  projectId: string,
): DomainRoleBinding | undefined {
  return bindings.find(
    (b) =>
      b.credentialId === credentialId &&
      b.projectId === projectId &&
      !b.agentId,
  )
}

/**
 * A credential is agent-bound when there is a RoleBinding with:
 *   credentialId === cred.id && agentId === agent.id
 */
function findAgentBinding(
  bindings: DomainRoleBinding[],
  credentialId: string,
  agentId: string,
): DomainRoleBinding | undefined {
  return bindings.find(
    (b) => b.credentialId === credentialId && b.agentId === agentId,
  )
}

/**
 * Inherited = project is bound but agent is NOT directly bound.
 */
function isInherited(
  bindings: DomainRoleBinding[],
  credentialId: string,
  agentId: string,
  projectId: string,
): boolean {
  return (
    !!findProjectBinding(bindings, credentialId, projectId) &&
    !findAgentBinding(bindings, credentialId, agentId)
  )
}

function globalColIndex(groups: ProjectGroup[], gIdx: number, colWithinGroup: number): number {
  let idx = 0
  for (let i = 0; i < gIdx; i++) {
    idx += 1 + groups[i].agents.length
  }
  return idx + colWithinGroup
}

function totalColumnCount(groups: ProjectGroup[]): number {
  return groups.reduce((sum, g) => sum + 1 + g.agents.length, 0)
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export function BindingMatrix({
  credentials,
  projects,
  agents,
  bindings,
  roleId,
}: BindingMatrixProps) {
  // --- filter / pagination state ---
  const [filterText, setFilterText] = useState('')
  const [selectedProjectFilter, setSelectedProjectFilter] = useState<string>(
    projects.length > 0 ? projects[0].id : '__all__',
  )
  const [currentPage, setCurrentPage] = useState(1)
  const [pendingCells, setPendingCells] = useState<Set<string>>(() => new Set())
  const [openColumnPopovers, setOpenColumnPopovers] = useState<Record<string, boolean>>({})
  const [openRowPopovers, setOpenRowPopovers] = useState<Record<string, boolean>>({})
  const [bulkConfirm, setBulkConfirm] = useState<BulkConfirmState>(INITIAL_BULK_CONFIRM)

  // Optimistic binding overlay: pending additions and deletions
  const [optimisticAdds, setOptimisticAdds] = useState<DomainRoleBinding[]>([])
  const [optimisticDeletes, setOptimisticDeletes] = useState<Set<string>>(() => new Set())

  // Merged bindings = server bindings + optimistic adds - optimistic deletes
  const effectiveBindings = useMemo(() => {
    const serverVisible = bindings.filter((b) => !optimisticDeletes.has(b.id))
    return [...serverVisible, ...optimisticAdds]
  }, [bindings, optimisticAdds, optimisticDeletes])

  // Refs for focus management
  const focusCellRef = useRef<HTMLButtonElement | null>(null)

  // Mutations
  const createBinding = useCreateRoleBinding()
  const deleteBinding = useDeleteRoleBinding()

  // --- Reset page when filter changes ---
  useEffect(() => {
    setCurrentPage(1)
  }, [filterText, selectedProjectFilter])

  // Sync selectedProjectFilter when projects change
  useEffect(() => {
    if (projects.length > 0 && selectedProjectFilter === '__all__') {
      setSelectedProjectFilter(projects[0].id)
    }
  }, [projects, selectedProjectFilter])

  // --- Build project groups ---
  const allProjectGroups = useMemo<ProjectGroup[]>(() => {
    const sorted = [...projects].sort((a, b) => a.name.localeCompare(b.name))
    return sorted.map((p) => ({
      project: p,
      agents: agents
        .filter((a) => a.projectId === p.id)
        .sort((a, b) => a.name.localeCompare(b.name)),
    }))
  }, [projects, agents])

  const projectGroups = useMemo<ProjectGroup[]>(() => {
    if (selectedProjectFilter === '__all__') return allProjectGroups
    return allProjectGroups.filter((g) => g.project.id === selectedProjectFilter)
  }, [allProjectGroups, selectedProjectFilter])

  const hasAnyAgents = useMemo(
    () => projectGroups.some((g) => g.agents.length > 0),
    [projectGroups],
  )

  const totalCols = useMemo(() => totalColumnCount(projectGroups), [projectGroups])

  // --- Filtered & paginated credentials ---
  const filteredCredentials = useMemo(() => {
    const q = filterText.trim().toLowerCase()
    const sorted = [...credentials].sort((a, b) => a.name.localeCompare(b.name))
    if (!q) return sorted
    return sorted.filter((c) => c.name.toLowerCase().includes(q))
  }, [credentials, filterText])

  const totalPages = Math.ceil(filteredCredentials.length / PAGE_SIZE)
  const startRow = filteredCredentials.length === 0 ? 0 : (currentPage - 1) * PAGE_SIZE + 1
  const endRow = Math.min(currentPage * PAGE_SIZE, filteredCredentials.length)

  const paginatedCredentials = useMemo(
    () => filteredCredentials.slice((currentPage - 1) * PAGE_SIZE, currentPage * PAGE_SIZE),
    [filteredCredentials, currentPage],
  )

  // --- Helper to add/remove pending cell ---
  const addPending = useCallback((key: string) => {
    setPendingCells((prev) => {
      const next = new Set(prev)
      next.add(key)
      return next
    })
  }, [])

  const removePending = useCallback((key: string) => {
    setPendingCells((prev) => {
      const next = new Set(prev)
      next.delete(key)
      return next
    })
  }, [])

  // --- Close column/row popovers ---
  const closeColumnPopover = useCallback((id: string) => {
    setOpenColumnPopovers((prev) => ({ ...prev, [id]: false }))
  }, [])

  const closeRowPopover = useCallback((id: string) => {
    setOpenRowPopovers((prev) => ({ ...prev, [id]: false }))
  }, [])

  // --- Toggle a single cell ---
  const toggleCell = useCallback(
    async (params: {
      credentialId: string
      targetId: string
      targetType: 'project' | 'agent'
      projectId?: string
    }) => {
      const key = cellKey(params.credentialId, params.targetId)
      if (pendingCells.has(key)) return

      const existingBinding =
        params.targetType === 'project'
          ? findProjectBinding(effectiveBindings, params.credentialId, params.targetId)
          : findAgentBinding(effectiveBindings, params.credentialId, params.targetId)

      addPending(key)

      if (existingBinding) {
        // Optimistic delete
        setOptimisticDeletes((prev) => {
          const next = new Set(prev)
          next.add(existingBinding.id)
          return next
        })
        try {
          await deleteBinding.mutateAsync(existingBinding.id)
        } catch {
          // Rollback
          setOptimisticDeletes((prev) => {
            const next = new Set(prev)
            next.delete(existingBinding.id)
            return next
          })
        } finally {
          // Clean up optimistic delete once server data refreshes
          setOptimisticDeletes((prev) => {
            const next = new Set(prev)
            next.delete(existingBinding.id)
            return next
          })
          removePending(key)
        }
      } else {
        // Optimistic add — create a temporary binding
        const tempId = `temp-${Date.now()}-${Math.random().toString(36).slice(2, 9)}`
        const tempBinding: DomainRoleBinding = {
          id: tempId,
          roleId,
          scope: 'credential',
          userId: null,
          projectId: params.targetType === 'project' ? params.targetId : (params.projectId ?? null),
          agentId: params.targetType === 'agent' ? params.targetId : null,
          credentialId: params.credentialId,
          sessionId: null,
          createdAt: new Date().toISOString(),
          updatedAt: new Date().toISOString(),
        }
        setOptimisticAdds((prev) => [...prev, tempBinding])
        try {
          await createBinding.mutateAsync({
            roleId,
            scope: 'credential',
            credentialId: params.credentialId,
            projectId: params.targetType === 'project' ? params.targetId : params.projectId,
            agentId: params.targetType === 'agent' ? params.targetId : undefined,
          })
        } catch {
          // Rollback
        } finally {
          setOptimisticAdds((prev) => prev.filter((b) => b.id !== tempId))
          removePending(key)
        }
      }
    },
    [pendingCells, effectiveBindings, addPending, removePending, roleId, createBinding, deleteBinding],
  )

  // --- Batch toggle ---
  const batchToggle = useCallback(
    async (calls: Array<Parameters<typeof toggleCell>[0]>) => {
      for (let i = 0; i < calls.length; i += BATCH_CHUNK_SIZE) {
        const chunk = calls.slice(i, i + BATCH_CHUNK_SIZE)
        await Promise.all(chunk.map((c) => toggleCell(c)))
      }
    },
    [toggleCell],
  )

  // --- Bulk operations ---
  const bulkBindProject = useCallback(
    async (projectId: string) => {
      closeColumnPopover(projectId)
      const unboundCreds = credentials.filter(
        (c) => !findProjectBinding(effectiveBindings, c.id, projectId),
      )
      await batchToggle(
        unboundCreds.map((cred) => ({
          credentialId: cred.id,
          targetId: projectId,
          targetType: 'project' as const,
        })),
      )
    },
    [credentials, effectiveBindings, batchToggle, closeColumnPopover],
  )

  const bulkUnbindProject = useCallback(
    (projectId: string) => {
      const boundCreds = credentials.filter(
        (c) => !!findProjectBinding(effectiveBindings, c.id, projectId),
      )
      const project = projects.find((p) => p.id === projectId)
      const projectName = project?.name ?? projectId
      closeColumnPopover(projectId)

      setBulkConfirm({
        show: true,
        title: 'Unbind all credentials from project',
        message: `This will unbind ${boundCreds.length} credential${boundCreds.length === 1 ? '' : 's'} from ${projectName}. Agents in this project will lose access. Continue?`,
        count: boundCreds.length,
        onConfirm: () => {
          void batchToggle(
            boundCreds.map((cred) => ({
              credentialId: cred.id,
              targetId: projectId,
              targetType: 'project' as const,
            })),
          )
        },
      })
    },
    [credentials, effectiveBindings, projects, batchToggle, closeColumnPopover],
  )

  const bulkBindAgent = useCallback(
    async (agentId: string, projectId: string) => {
      closeColumnPopover(agentId)
      const unboundCreds = credentials.filter(
        (c) => !findAgentBinding(effectiveBindings, c.id, agentId),
      )
      await batchToggle(
        unboundCreds.map((cred) => ({
          credentialId: cred.id,
          targetId: agentId,
          targetType: 'agent' as const,
          projectId,
        })),
      )
    },
    [credentials, effectiveBindings, batchToggle, closeColumnPopover],
  )

  const bulkUnbindAgent = useCallback(
    (agentId: string) => {
      const boundCreds = credentials.filter(
        (c) => !!findAgentBinding(effectiveBindings, c.id, agentId),
      )
      const agent = agents.find((a) => a.id === agentId)
      const agentName = agent?.name ?? agentId
      closeColumnPopover(agentId)

      setBulkConfirm({
        show: true,
        title: 'Unbind all credentials from agent',
        message: `This will unbind ${boundCreds.length} credential${boundCreds.length === 1 ? '' : 's'} from agent ${agentName}. Continue?`,
        count: boundCreds.length,
        onConfirm: () => {
          void batchToggle(
            boundCreds.map((cred) => ({
              credentialId: cred.id,
              targetId: agentId,
              targetType: 'agent' as const,
            })),
          )
        },
      })
    },
    [credentials, effectiveBindings, agents, batchToggle, closeColumnPopover],
  )

  const bulkBindRowProjects = useCallback(
    async (cred: DomainCredential) => {
      closeRowPopover(cred.id)
      const unboundProjects = projects.filter(
        (p) => !findProjectBinding(effectiveBindings, cred.id, p.id),
      )
      await batchToggle(
        unboundProjects.map((p) => ({
          credentialId: cred.id,
          targetId: p.id,
          targetType: 'project' as const,
        })),
      )
    },
    [projects, effectiveBindings, batchToggle, closeRowPopover],
  )

  const bulkUnbindRow = useCallback(
    (cred: DomainCredential) => {
      closeRowPopover(cred.id)
      const calls: Array<Parameters<typeof toggleCell>[0]> = []
      for (const p of projects) {
        if (findProjectBinding(effectiveBindings, cred.id, p.id)) {
          calls.push({
            credentialId: cred.id,
            targetId: p.id,
            targetType: 'project' as const,
          })
        }
      }
      for (const a of agents) {
        if (findAgentBinding(effectiveBindings, cred.id, a.id)) {
          calls.push({
            credentialId: cred.id,
            targetId: a.id,
            targetType: 'agent' as const,
            projectId: a.projectId ?? undefined,
          })
        }
      }

      setBulkConfirm({
        show: true,
        title: 'Remove all access for credential',
        message: `This will remove all access for ${cred.name}. Continue?`,
        count: calls.length,
        onConfirm: () => {
          void batchToggle(calls)
        },
      })
    },
    [projects, agents, effectiveBindings, batchToggle, closeRowPopover, toggleCell],
  )

  // --- Keyboard navigation ---
  const handleCellKeydown = useCallback(
    (event: React.KeyboardEvent<HTMLButtonElement>, row: number, col: number) => {
      let targetRow = row
      let targetCol = col
      switch (event.key) {
        case 'ArrowUp':
          targetRow = row - 1
          break
        case 'ArrowDown':
          targetRow = row + 1
          break
        case 'ArrowLeft':
          targetCol = col - 1
          break
        case 'ArrowRight':
          targetCol = col + 1
          break
        default:
          return
      }
      event.preventDefault()
      const next = document.querySelector<HTMLElement>(
        `[data-matrix-row="${targetRow}"][data-matrix-col="${targetCol}"]`,
      )
      if (next) next.focus()
    },
    [],
  )

  // --- Render helpers ---
  const renderProjectHeaderPopover = useCallback(
    (group: ProjectGroup) => (
      <Popover
        open={openColumnPopovers[group.project.id] ?? false}
        onOpenChange={(open) =>
          setOpenColumnPopovers((prev) => ({ ...prev, [group.project.id]: open }))
        }
      >
        <PopoverTrigger asChild>
          <button
            type="button"
            className="group px-3 py-2 cursor-pointer hover:bg-accent/60 rounded-sm transition-colors whitespace-nowrap font-semibold text-sm inline-flex items-center gap-1"
          >
            {group.project.name}
            <ChevronDown className="h-3 w-3 text-muted-foreground opacity-50 group-hover:opacity-100 transition-opacity" />
          </button>
        </PopoverTrigger>
        <PopoverContent className="w-56 p-2" align="start">
          <div className="space-y-1">
            <p className="text-xs font-medium text-muted-foreground px-2 py-1">
              Project: {group.project.name}
            </p>
            <Separator />
            <Button
              variant="ghost"
              size="sm"
              className="w-full justify-start text-sm"
              onClick={() => void bulkBindProject(group.project.id)}
            >
              Bind all credentials
            </Button>
            <Button
              variant="destructive"
              size="sm"
              className="w-full justify-start text-sm"
              onClick={() => bulkUnbindProject(group.project.id)}
            >
              Unbind all credentials
            </Button>
          </div>
        </PopoverContent>
      </Popover>
    ),
    [openColumnPopovers, bulkBindProject, bulkUnbindProject],
  )

  const renderAgentHeaderPopover = useCallback(
    (agent: DomainAgent, group: ProjectGroup) => (
      <Popover
        open={openColumnPopovers[agent.id] ?? false}
        onOpenChange={(open) =>
          setOpenColumnPopovers((prev) => ({ ...prev, [agent.id]: open }))
        }
      >
        <PopoverTrigger asChild>
          <button
            type="button"
            className="group flex items-end justify-center h-full w-full pb-1 cursor-pointer hover:bg-accent/60 rounded-sm transition-colors"
          >
            <Tooltip>
              <TooltipTrigger asChild>
                <span
                  className="text-xs whitespace-nowrap"
                  style={{
                    writingMode: 'vertical-lr',
                    textOrientation: 'mixed',
                    display: 'inline-block',
                  }}
                >
                  {agent.displayName ?? agent.name}
                </span>
              </TooltipTrigger>
              <TooltipContent>
                <p>Agent: {group.project.name}/{agent.name}</p>
              </TooltipContent>
            </Tooltip>
          </button>
        </PopoverTrigger>
        <PopoverContent className="w-56 p-2" align="start">
          <div className="space-y-1">
            <p className="text-xs font-medium text-muted-foreground px-2 py-1">
              Agent: {agent.displayName ?? agent.name}
            </p>
            <Separator />
            <Button
              variant="ghost"
              size="sm"
              className="w-full justify-start text-sm"
              onClick={() => void bulkBindAgent(agent.id, group.project.id)}
            >
              Bind all credentials to this agent
            </Button>
            <Button
              variant="destructive"
              size="sm"
              className="w-full justify-start text-sm"
              onClick={() => bulkUnbindAgent(agent.id)}
            >
              Unbind all credentials from this agent
            </Button>
          </div>
        </PopoverContent>
      </Popover>
    ),
    [openColumnPopovers, bulkBindAgent, bulkUnbindAgent],
  )

  return (
    <TooltipProvider delayDuration={300}>
      <div className="space-y-3">
        {/* --- Filters: project dropdown + credential name search --- */}
        <div className="flex items-center justify-between gap-4">
          <Select value={selectedProjectFilter} onValueChange={setSelectedProjectFilter}>
            <SelectTrigger className="w-[200px]">
              <SelectValue placeholder="All projects" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="__all__">All projects</SelectItem>
              {projects.map((p) => (
                <SelectItem key={p.id} value={p.id}>
                  {p.name}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <div className="relative w-64">
            <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
            <Input
              value={filterText}
              onChange={(e) => setFilterText(e.target.value)}
              placeholder="Filter credentials..."
              className="pl-9 h-9"
            />
          </div>
        </div>

        {/* --- Warning banner when showing too many columns --- */}
        {selectedProjectFilter === '__all__' && totalCols > 30 && (
          <div className="flex items-center gap-2 rounded-md border border-amber-500/50 bg-amber-50 dark:bg-amber-950/30 px-4 py-2 text-sm text-amber-800 dark:text-amber-200">
            <AlertTriangle className="h-4 w-4 shrink-0" />
            <span>Showing all {totalCols} columns. Select a specific project for easier editing.</span>
          </div>
        )}

        {/* --- Legend bar --- */}
        <div className="flex items-center gap-4 text-xs text-muted-foreground">
          <div className="flex items-center gap-1.5">
            <span className="inline-block h-3.5 w-3.5 rounded-sm bg-green-600" />
            <span>Directly bound</span>
          </div>
          <div className="flex items-center gap-1.5">
            <span className="inline-block h-3.5 w-3.5 rounded-sm bg-green-600/40" />
            <span>Inherited from project</span>
          </div>
          <div className="flex items-center gap-1.5">
            <span className="inline-block h-3.5 w-3.5 rounded-sm border-2 border-muted-foreground/30" />
            <span>Not bound</span>
          </div>
        </div>

        {/* --- Matrix table --- */}
        <div className="overflow-auto max-h-[70vh]">
          <Table>
            <TableHeader className="sticky top-0 z-20 bg-background">
              {!hasAnyAgents ? (
                /* === SIMPLE LAYOUT: no agents anywhere === */
                <TableRow>
                  <TableHead className="sticky left-0 z-30 bg-background min-w-[160px]">
                    Credential
                  </TableHead>
                  {projectGroups.map((group, gIdx) => (
                    <TableHead
                      key={group.project.id}
                      className={cn(
                        'text-center p-0',
                        gIdx > 0 && 'border-l border-l-border',
                        globalColIndex(projectGroups, gIdx, 0) % 2 === 1 && 'bg-muted/20',
                      )}
                    >
                      {renderProjectHeaderPopover(group)}
                    </TableHead>
                  ))}
                </TableRow>
              ) : (
                /* === HIERARCHICAL LAYOUT: two header rows === */
                <>
                  {/* Row 1: project names spanning their agent columns */}
                  <TableRow>
                    <TableHead
                      rowSpan={2}
                      className="sticky left-0 z-30 bg-background min-w-[160px]"
                    >
                      Credential
                    </TableHead>
                    {projectGroups.map((group, gIdx) =>
                      group.agents.length > 0 ? (
                        <TableHead
                          key={group.project.id}
                          colSpan={1 + group.agents.length}
                          className={cn(
                            'text-center font-semibold text-sm text-muted-foreground border-b-2 border-primary/30 p-0',
                            gIdx > 0 && 'border-l-2 border-l-border',
                          )}
                        >
                          {renderProjectHeaderPopover(group)}
                        </TableHead>
                      ) : (
                        <TableHead
                          key={group.project.id}
                          rowSpan={2}
                          className={cn(
                            'text-center p-0',
                            gIdx > 0 && 'border-l border-l-border',
                            globalColIndex(projectGroups, gIdx, 0) % 2 === 1 && 'bg-muted/20',
                          )}
                        >
                          {renderProjectHeaderPopover(group)}
                        </TableHead>
                      ),
                    )}
                  </TableRow>

                  {/* Row 2: "All" column + agent sub-columns */}
                  <TableRow className="h-[120px]">
                    {projectGroups.map((group, gIdx) =>
                      group.agents.length > 0 ? (
                        <AgentSubHeaders
                          key={group.project.id}
                          group={group}
                          gIdx={gIdx}
                          projectGroups={projectGroups}
                          renderAgentHeaderPopover={renderAgentHeaderPopover}
                        />
                      ) : null,
                    )}
                  </TableRow>
                </>
              )}
            </TableHeader>

            <TableBody>
              {filteredCredentials.length > 0 ? (
                paginatedCredentials.map((cred, rowIndex) => (
                  <TableRow key={cred.id}>
                    {/* Row header: credential name with bulk-ops popover */}
                    <TableCell className="sticky left-0 z-10 bg-background font-medium border-r p-0">
                      <Popover
                        open={openRowPopovers[cred.id] ?? false}
                        onOpenChange={(open) =>
                          setOpenRowPopovers((prev) => ({ ...prev, [cred.id]: open }))
                        }
                      >
                        <PopoverTrigger asChild>
                          <button
                            type="button"
                            className="group w-full text-left px-4 py-2 cursor-pointer hover:bg-accent/60 rounded-sm transition-colors inline-flex items-center justify-between gap-1 max-w-[200px]"
                          >
                            <span className="truncate">{cred.name}</span>
                            <ChevronDown className="h-3 w-3 shrink-0 text-muted-foreground opacity-50 group-hover:opacity-100 transition-opacity" />
                          </button>
                        </PopoverTrigger>
                        <PopoverContent className="w-56 p-2" align="start" side="right">
                          <div className="space-y-1">
                            <p className="text-xs font-medium text-muted-foreground px-2 py-1">
                              {cred.name}
                            </p>
                            <Separator />
                            {projectGroups.length > 0 && (
                              <Button
                                variant="ghost"
                                size="sm"
                                className="w-full justify-start text-sm"
                                onClick={() => void bulkBindRowProjects(cred)}
                              >
                                Bind to all projects
                              </Button>
                            )}
                            <Button
                              variant="destructive"
                              size="sm"
                              className="w-full justify-start text-sm"
                              onClick={() => bulkUnbindRow(cred)}
                            >
                              Unbind from all
                            </Button>
                          </div>
                        </PopoverContent>
                      </Popover>
                    </TableCell>

                    {/* Cells per project group */}
                    {projectGroups.map((group, gIdx) => (
                      <GroupCells
                        key={group.project.id}
                        group={group}
                        gIdx={gIdx}
                        cred={cred}
                        rowIndex={rowIndex}
                        projectGroups={projectGroups}
                        hasAnyAgents={hasAnyAgents}
                        effectiveBindings={effectiveBindings}
                        pendingCells={pendingCells}
                        onToggle={toggleCell}
                        onKeyDown={handleCellKeydown}
                        focusCellRef={focusCellRef}
                      />
                    ))}
                  </TableRow>
                ))
              ) : (
                <TableRow>
                  <TableCell
                    colSpan={totalCols + 1}
                    className="h-24 text-center text-muted-foreground"
                  >
                    {filterText.trim()
                      ? `No credentials match "${filterText.trim()}".`
                      : 'No credentials to display. Create credentials to manage bindings.'}
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        </div>

        {/* --- Pagination controls --- */}
        {filteredCredentials.length > PAGE_SIZE && (
          <div className="flex items-center justify-between py-2">
            <span className="text-sm text-muted-foreground">
              Showing {startRow}-{endRow} of {filteredCredentials.length}
            </span>
            <div className="flex gap-2">
              <Button
                variant="outline"
                size="sm"
                disabled={currentPage <= 1}
                onClick={() => setCurrentPage((p) => p - 1)}
              >
                Previous
              </Button>
              <span className="text-sm text-muted-foreground flex items-center px-2">
                Page {currentPage} of {totalPages}
              </span>
              <Button
                variant="outline"
                size="sm"
                disabled={currentPage >= totalPages}
                onClick={() => setCurrentPage((p) => p + 1)}
              >
                Next
              </Button>
            </div>
          </div>
        )}

        {/* --- Bulk unbind confirmation dialog --- */}
        <AlertDialog
          open={bulkConfirm.show}
          onOpenChange={(open) => {
            if (!open) setBulkConfirm(INITIAL_BULK_CONFIRM)
          }}
        >
          <AlertDialogContent>
            <AlertDialogHeader>
              <AlertDialogTitle>{bulkConfirm.title}</AlertDialogTitle>
              <AlertDialogDescription>{bulkConfirm.message}</AlertDialogDescription>
            </AlertDialogHeader>
            <AlertDialogFooter>
              <AlertDialogCancel onClick={() => setBulkConfirm(INITIAL_BULK_CONFIRM)}>
                Cancel
              </AlertDialogCancel>
              <AlertDialogAction
                className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
                onClick={() => {
                  const { onConfirm } = bulkConfirm
                  setBulkConfirm(INITIAL_BULK_CONFIRM)
                  onConfirm()
                }}
              >
                Unbind {bulkConfirm.count} credential{bulkConfirm.count === 1 ? '' : 's'}
              </AlertDialogAction>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialog>
      </div>
    </TooltipProvider>
  )
}

// ---------------------------------------------------------------------------
// Sub-components (extracted for readability, avoid JSX in map callbacks)
// ---------------------------------------------------------------------------

function AgentSubHeaders({
  group,
  gIdx,
  projectGroups,
  renderAgentHeaderPopover,
}: {
  group: ProjectGroup
  gIdx: number
  projectGroups: ProjectGroup[]
  renderAgentHeaderPopover: (agent: DomainAgent, group: ProjectGroup) => React.ReactNode
}) {
  return (
    <>
      {/* "All" column for project-level binding */}
      <TableHead
        className={cn(
          'text-center min-w-[36px] w-[36px] align-bottom p-0',
          gIdx > 0 && 'border-l-2 border-l-border',
          globalColIndex(projectGroups, gIdx, 0) % 2 === 1 && 'bg-muted/20',
        )}
      >
        <Tooltip>
          <TooltipTrigger asChild>
            <span className="flex items-end justify-center h-full w-full pb-2">
              <span className="text-xs font-semibold text-muted-foreground/60">All</span>
            </span>
          </TooltipTrigger>
          <TooltipContent>
            <p>Project-level binding: {group.project.name}</p>
          </TooltipContent>
        </Tooltip>
      </TableHead>

      {/* Agent sub-columns */}
      {group.agents.map((agent, aIdx) => (
        <TableHead
          key={agent.id}
          className={cn(
            'text-center min-w-[56px] align-bottom p-0',
            globalColIndex(projectGroups, gIdx, aIdx + 1) % 2 === 1 && 'bg-muted/20',
          )}
        >
          {renderAgentHeaderPopover(agent, group)}
        </TableHead>
      ))}
    </>
  )
}

function GroupCells({
  group,
  gIdx,
  cred,
  rowIndex,
  projectGroups,
  hasAnyAgents,
  effectiveBindings,
  pendingCells,
  onToggle,
  onKeyDown,
  focusCellRef,
}: {
  group: ProjectGroup
  gIdx: number
  cred: DomainCredential
  rowIndex: number
  projectGroups: ProjectGroup[]
  hasAnyAgents: boolean
  effectiveBindings: DomainRoleBinding[]
  pendingCells: Set<string>
  onToggle: (params: {
    credentialId: string
    targetId: string
    targetType: 'project' | 'agent'
    projectId?: string
  }) => void
  onKeyDown: (event: React.KeyboardEvent<HTMLButtonElement>, row: number, col: number) => void
  focusCellRef: React.MutableRefObject<HTMLButtonElement | null>
}) {
  const projectBound = !!findProjectBinding(effectiveBindings, cred.id, group.project.id)
  const projectPending = pendingCells.has(cellKey(cred.id, group.project.id))
  const colIdx = globalColIndex(projectGroups, gIdx, 0)

  return (
    <>
      {/* Project-level binding cell */}
      <TableCell
        className={cn(
          'text-center p-0',
          hasAnyAgents && gIdx > 0 && group.agents.length > 0 && 'border-l-2 border-border',
          gIdx > 0 && (!hasAnyAgents || group.agents.length === 0) && 'border-l border-border',
          colIdx % 2 === 1 && 'bg-muted/20',
        )}
      >
        <Tooltip>
          <TooltipTrigger asChild>
            <button
              ref={focusCellRef}
              type="button"
              className="h-8 w-8 flex items-center justify-center mx-auto cursor-pointer rounded transition-colors hover:bg-accent/50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-1 disabled:pointer-events-none disabled:opacity-50"
              disabled={projectPending}
              data-matrix-row={rowIndex}
              data-matrix-col={colIdx}
              onKeyDown={(e) => onKeyDown(e, rowIndex, colIdx)}
              onClick={() =>
                onToggle({
                  credentialId: cred.id,
                  targetId: group.project.id,
                  targetType: 'project',
                })
              }
            >
              {projectPending ? (
                <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />
              ) : projectBound ? (
                <span className="h-4 w-4 rounded-sm bg-green-600 flex items-center justify-center">
                  <Check className="h-3 w-3 text-white" />
                </span>
              ) : (
                <span className="h-4 w-4 rounded-sm border-2 border-muted-foreground/30 inline-block" />
              )}
            </button>
          </TooltipTrigger>
          <TooltipContent>
            <p>
              {projectBound ? 'Unbind from' : 'Bind to'} project: {group.project.name}
            </p>
          </TooltipContent>
        </Tooltip>
      </TableCell>

      {/* Agent cells */}
      {group.agents.map((agent, aIdx) => {
        const agentColIdx = globalColIndex(projectGroups, gIdx, aIdx + 1)
        const inherited = isInherited(effectiveBindings, cred.id, agent.id, group.project.id)
        const agentBound = !!findAgentBinding(effectiveBindings, cred.id, agent.id)
        const agentPending = pendingCells.has(cellKey(cred.id, agent.id))

        return (
          <TableCell
            key={agent.id}
            className={cn(
              'text-center p-0',
              agentColIdx % 2 === 1 && 'bg-muted/20',
            )}
          >
            {inherited ? (
              /* Inherited state: non-clickable, manage at project level */
              <Tooltip>
                <TooltipTrigger asChild>
                  <span
                    className="h-8 w-8 flex items-center justify-center mx-auto rounded opacity-70"
                    data-matrix-row={rowIndex}
                    data-matrix-col={agentColIdx}
                  >
                    <span className="h-4 w-4 rounded-sm bg-green-600/40 flex items-center justify-center">
                      <Check className="h-3 w-3 text-white/80" />
                    </span>
                  </span>
                </TooltipTrigger>
                <TooltipContent>
                  <p>Inherited from project. Manage at project level.</p>
                </TooltipContent>
              </Tooltip>
            ) : (
              /* Direct binding or unbound state */
              <Tooltip>
                <TooltipTrigger asChild>
                  <button
                    type="button"
                    className="h-8 w-8 flex items-center justify-center mx-auto cursor-pointer rounded transition-colors hover:bg-accent/50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-1 disabled:pointer-events-none disabled:opacity-50"
                    disabled={agentPending}
                    data-matrix-row={rowIndex}
                    data-matrix-col={agentColIdx}
                    onKeyDown={(e) => onKeyDown(e, rowIndex, agentColIdx)}
                    onClick={() =>
                      onToggle({
                        credentialId: cred.id,
                        targetId: agent.id,
                        targetType: 'agent',
                        projectId: group.project.id,
                      })
                    }
                  >
                    {agentPending ? (
                      <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />
                    ) : agentBound ? (
                      <span className="h-4 w-4 rounded-sm bg-green-600 flex items-center justify-center">
                        <Check className="h-3 w-3 text-white" />
                      </span>
                    ) : (
                      <span className="h-4 w-4 rounded-sm border-2 border-muted-foreground/30 inline-block" />
                    )}
                  </button>
                </TooltipTrigger>
                <TooltipContent>
                  <p>
                    {agentBound ? 'Unbind from' : 'Bind to'} agent: {agent.displayName ?? agent.name}
                  </p>
                </TooltipContent>
              </Tooltip>
            )}
          </TableCell>
        )
      })}
    </>
  )
}
