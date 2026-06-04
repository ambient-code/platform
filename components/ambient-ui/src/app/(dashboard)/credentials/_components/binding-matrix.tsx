'use client'

import {
  useState,
  useMemo,
  useCallback,
  useRef,
  useEffect,
} from 'react'
import { Check, ChevronDown, Loader2, Search, X, AlertTriangle, Link2, Unlink, MoreVertical, Plus, Minus, ShieldCheck, ShieldOff, KeyRound, FolderOpen, Settings2 } from 'lucide-react'
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
import { Badge } from '@/components/ui/badge'
import { Separator } from '@/components/ui/separator'
import { toast } from 'sonner'
import { useCreateRoleBinding, useDeleteRoleBinding } from '@/queries/use-role-bindings'
import { cn } from '@/lib/utils'
import type { DomainCredential, DomainRoleBinding, DomainProject, DomainAgent } from '@/domain/types'
import { cellKey, findProjectBinding, findAgentBinding, isInherited } from './binding-helpers'

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

type ProjectGroup = {
  project: DomainProject
  agents: DomainAgent[]
}

type BulkConfirmGroup = {
  label: string
  children: string[]
}

type BulkConfirmDetails = {
  context: string[]
  items: string[]
  itemLabel?: string
  groups?: BulkConfirmGroup[]
}

type BulkConfirmState = {
  show: boolean
  title: string
  message: React.ReactNode
  details?: BulkConfirmDetails
  count: number
  variant: 'grant' | 'revoke'
  confirmLabel: string
  onConfirm: () => void
}

type BindingMatrixProps = {
  credentials: DomainCredential[]
  projects: DomainProject[]
  agents: DomainAgent[]
  bindings: DomainRoleBinding[]
  roleId: string
  initialFilter?: string
  onEditCredential?: (credential: DomainCredential) => void
}

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

const PAGE_SIZE = 25
const BATCH_CONCURRENCY = 20

const INITIAL_BULK_CONFIRM: BulkConfirmState = {
  show: false,
  title: '',
  message: '',
  count: 0,
  variant: 'grant',
  confirmLabel: '',
  onConfirm: () => undefined,
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

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
  initialFilter,
  onEditCredential,
}: BindingMatrixProps) {
  // --- filter / pagination state ---
  const [filterText, setFilterText] = useState(initialFilter ?? '')
  const [selectedProjectFilter, setSelectedProjectFilter] = useState<string>('__all__')
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

  // --- Sync filter from parent ---
  useEffect(() => {
    if (initialFilter !== undefined) setFilterText(initialFilter)
  }, [initialFilter])

  // --- Reset page when filter changes ---
  useEffect(() => {
    setCurrentPage(1)
  }, [filterText, selectedProjectFilter])


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
          setOptimisticDeletes((prev) => {
            const next = new Set(prev)
            next.delete(existingBinding.id)
            return next
          })
          toast.error('Failed to remove binding. Please try again.')
        } finally {
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
          toast.error('Failed to create binding. Please try again.')
        } finally {
          setOptimisticAdds((prev) => prev.filter((b) => b.id !== tempId))
          removePending(key)
        }
      }
    },
    [pendingCells, effectiveBindings, addPending, removePending, roleId, createBinding, deleteBinding],
  )

  // --- Batch toggle with concurrency pool ---
  const batchToggle = useCallback(
    async (calls: Array<Parameters<typeof toggleCell>[0]>) => {
      if (calls.length === 0) return
      const toastId = calls.length > 3
        ? toast.loading(`Updating ${calls.length} bindings...`)
        : undefined

      const executing = new Set<Promise<void>>()
      for (const call of calls) {
        const p = toggleCell(call).finally(() => executing.delete(p))
        executing.add(p)
        if (executing.size >= BATCH_CONCURRENCY) {
          await Promise.race(executing)
        }
      }
      await Promise.all(executing)

      if (toastId) {
        toast.success(`Updated ${calls.length} bindings`, { id: toastId })
      }
    },
    [toggleCell],
  )

  // --- Bulk operations ---
  const bulkBindProject = useCallback(
    (projectId: string) => {
      const unboundCreds = credentials.filter(
        (c) => !findProjectBinding(effectiveBindings, c.id, projectId),
      )
      if (unboundCreds.length === 0) { closeColumnPopover(projectId); return }
      const project = projects.find((p) => p.id === projectId)
      const projectName = project?.name ?? projectId
      const group = allProjectGroups.find((g) => g.project.id === projectId)
      const agentNames = group?.agents.map((a) => a.displayName ?? a.name) ?? []
      closeColumnPopover(projectId)

      setBulkConfirm({
        show: true,
        title: `Grant access to "${projectName}"`,
        message: <><span className="font-semibold text-emerald-600 dark:text-emerald-400">{unboundCreds.length} credential{unboundCreds.length === 1 ? '' : 's'}</span> will gain project-wide access{agentNames.length > 0 ? <>, inherited by <span className="font-semibold text-emerald-600 dark:text-emerald-400">{agentNames.length} agent{agentNames.length === 1 ? '' : 's'}</span></> : ''}.</>,
        details: {
          context: agentNames.length > 0 ? [`Agents: ${agentNames.join(', ')}`] : [],
          items: unboundCreds.map((c) => c.name),
          itemLabel: 'credential',
        },
        count: unboundCreds.length,
        variant: 'grant',
        confirmLabel: `Grant ${unboundCreds.length} credential${unboundCreds.length === 1 ? '' : 's'}`,
        onConfirm: () => {
          void batchToggle(
            unboundCreds.map((cred) => ({
              credentialId: cred.id,
              targetId: projectId,
              targetType: 'project' as const,
            })),
          )
        },
      })
    },
    [credentials, effectiveBindings, projects, allProjectGroups, batchToggle, closeColumnPopover],
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
        title: `Revoke project-wide access from "${projectName}"`,
        message: <><span className="font-semibold text-destructive">{boundCreds.length} credential{boundCreds.length === 1 ? '' : 's'}</span> will lose project-wide access. Agents with direct bindings keep their access.</>,
        details: {
          context: [],
          items: boundCreds.map((c) => c.name),
          itemLabel: 'credential',
        },
        count: boundCreds.length,
        variant: 'revoke',
        confirmLabel: `Revoke ${boundCreds.length} credential${boundCreds.length === 1 ? '' : 's'}`,
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
    (agentId: string, projectId: string) => {
      const unboundCreds = credentials.filter(
        (c) => !findAgentBinding(effectiveBindings, c.id, agentId),
      )
      if (unboundCreds.length === 0) { closeColumnPopover(agentId); return }
      const agent = agents.find((a) => a.id === agentId)
      const agentName = agent?.displayName ?? agent?.name ?? agentId
      closeColumnPopover(agentId)

      setBulkConfirm({
        show: true,
        title: `Grant access to agent "${agentName}"`,
        message: <><span className="font-semibold text-emerald-600 dark:text-emerald-400">{unboundCreds.length} credential{unboundCreds.length === 1 ? '' : 's'}</span> will be directly bound to this agent.</>,
        details: {
          context: [],
          items: unboundCreds.map((c) => c.name),
          itemLabel: 'credential',
        },
        count: unboundCreds.length,
        variant: 'grant',
        confirmLabel: `Grant ${unboundCreds.length} credential${unboundCreds.length === 1 ? '' : 's'}`,
        onConfirm: () => {
          void batchToggle(
            unboundCreds.map((cred) => ({
              credentialId: cred.id,
              targetId: agentId,
              targetType: 'agent' as const,
              projectId,
            })),
          )
        },
      })
    },
    [credentials, agents, effectiveBindings, batchToggle, closeColumnPopover],
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
        title: `Revoke access from agent "${agentName}"`,
        message: <><span className="font-semibold text-destructive">{boundCreds.length} direct binding{boundCreds.length === 1 ? '' : 's'}</span> will be removed. Project-wide grants are not affected.</>,
        details: {
          context: [],
          items: boundCreds.map((c) => c.name),
          itemLabel: 'credential',
        },
        count: boundCreds.length,
        variant: 'revoke',
        confirmLabel: `Revoke ${boundCreds.length} credential${boundCreds.length === 1 ? '' : 's'}`,
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
    (cred: DomainCredential) => {
      const unboundProjects = projects.filter(
        (p) => !findProjectBinding(effectiveBindings, cred.id, p.id),
      )
      if (unboundProjects.length === 0) { closeRowPopover(cred.id); return }
      closeRowPopover(cred.id)

      const groups = unboundProjects.map((p) => {
        const group = allProjectGroups.find((g) => g.project.id === p.id)
        const agentNames = group?.agents.map((a) => a.displayName ?? a.name) ?? []
        return { label: p.name, children: agentNames }
      })

      setBulkConfirm({
        show: true,
        title: `Grant "${cred.name}" to all projects`,
        message: <>Project-wide access will be granted to <span className="font-semibold text-emerald-600 dark:text-emerald-400">{unboundProjects.length} project{unboundProjects.length === 1 ? '' : 's'}</span> and their agents.</>,
        details: {
          context: [],
          items: [],
          itemLabel: 'project',
          groups,
        },
        count: unboundProjects.length,
        variant: 'grant',
        confirmLabel: `Grant to ${unboundProjects.length} project${unboundProjects.length === 1 ? '' : 's'}`,
        onConfirm: () => {
          void batchToggle(
            unboundProjects.map((p) => ({
              credentialId: cred.id,
              targetId: p.id,
              targetType: 'project' as const,
            })),
          )
        },
      })
    },
    [projects, effectiveBindings, allProjectGroups, batchToggle, closeRowPopover],
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

      const boundProjects = projects.filter((p) => calls.some((c) => c.targetId === p.id))
      const boundAgentsByProject = new Map<string, string[]>()
      for (const a of agents) {
        if (calls.some((c) => c.targetId === a.id) && a.projectId) {
          const list = boundAgentsByProject.get(a.projectId) ?? []
          list.push(a.displayName ?? a.name)
          boundAgentsByProject.set(a.projectId, list)
        }
      }

      const groups: BulkConfirmGroup[] = []
      const projectsWithBindings = new Set(boundProjects.map((p) => p.id))
      for (const pid of new Set([...projectsWithBindings, ...boundAgentsByProject.keys()])) {
        const proj = projects.find((p) => p.id === pid)
        groups.push({
          label: proj?.name ?? pid,
          children: boundAgentsByProject.get(pid) ?? [],
        })
      }

      setBulkConfirm({
        show: true,
        title: `Revoke all access for "${cred.name}"`,
        message: <>All project and agent bindings will be removed (<span className="font-semibold text-destructive">{calls.length} total</span>).</>,
        details: {
          context: [],
          items: [],
          itemLabel: 'project',
          groups,
        },
        count: calls.length,
        variant: 'revoke',
        confirmLabel: `Revoke all (${calls.length})`,
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
            className="group px-3 py-2 cursor-pointer hover:bg-accent/60 rounded-sm transition-colors whitespace-nowrap font-semibold text-sm inline-flex items-center gap-1.5"
          >
            <FolderOpen className="h-3.5 w-3.5 text-muted-foreground" />
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
              <Plus className="h-3.5 w-3.5 mr-1.5" />
              Grant all to project
            </Button>
            <Button
              variant="destructive"
              size="sm"
              className="w-full justify-start text-sm"
              onClick={() => bulkUnbindProject(group.project.id)}
            >
              <Unlink className="h-3.5 w-3.5 mr-1.5" />
              Revoke from project
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
              <Plus className="h-3.5 w-3.5 mr-1.5" />
              Grant all to this agent
            </Button>
            <Button
              variant="destructive"
              size="sm"
              className="w-full justify-start text-sm"
              onClick={() => bulkUnbindAgent(agent.id)}
            >
              <Unlink className="h-3.5 w-3.5 mr-1.5" />
              Revoke from this agent
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
            <Search className={cn(
              'absolute left-2.5 top-2.5 h-4 w-4',
              filterText ? 'text-primary' : 'text-muted-foreground',
            )} />
            <Input
              value={filterText}
              onChange={(e) => setFilterText(e.target.value)}
              placeholder="Filter credentials..."
              className={cn(
                'pl-9 pr-8 h-9',
                filterText && 'ring-2 ring-primary/50 bg-primary/5',
              )}
            />
            {filterText && (
              <button
                type="button"
                onClick={() => setFilterText('')}
                className="absolute right-2 top-2 h-5 w-5 flex items-center justify-center rounded-sm text-muted-foreground hover:text-foreground hover:bg-accent"
                aria-label="Clear filter"
              >
                <X className="h-3.5 w-3.5" />
              </button>
            )}
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
            <span className="inline-block h-3.5 w-3.5 rounded-sm bg-matrix-bound" />
            <span>Directly bound</span>
          </div>
          {hasAnyAgents && (
            <div className="flex items-center gap-1.5">
              <span className="inline-block h-3.5 w-3.5 rounded-sm border-2 border-dashed border-matrix-bound/60 bg-matrix-bound/10" />
              <span>Inherited from project</span>
            </div>
          )}
          <div className="flex items-center gap-1.5">
            <span className="inline-block h-3.5 w-3.5 rounded-sm border-2 border-muted-foreground/30" />
            <span>Not bound</span>
          </div>
        </div>

        {/* --- Filtered state indicator --- */}
        {filterText && (
          <div className="flex items-center gap-2 text-sm text-muted-foreground">
            <span>
              Filtered to <span className="font-medium text-foreground">{filteredCredentials.length}</span> of {credentials.length} credentials
            </span>
            <span className="text-muted-foreground/40">·</span>
            <button
              type="button"
              onClick={() => setFilterText('')}
              className="text-primary hover:text-primary/80 text-sm"
            >
              Show all
            </button>
          </div>
        )}

        {/* --- Axis label --- */}
        <div className="flex items-end justify-between">
          <span className="text-[11px] font-medium uppercase tracking-wider text-muted-foreground/60">
            Credentials
          </span>
          <span className="text-[11px] font-medium uppercase tracking-wider text-muted-foreground/60">
            {hasAnyAgents ? 'Projects & Agents' : 'Projects'}
          </span>
        </div>

        {/* --- Matrix table --- */}
        <div className="overflow-auto max-h-[70vh]">
          <Table>
            <TableHeader className="sticky top-0 z-20 bg-background">
              {!hasAnyAgents ? (
                /* === SIMPLE LAYOUT: no agents anywhere === */
                <TableRow>
                  <TableHead className="sticky left-0 z-30 bg-background min-w-[200px]" />
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
                      className="sticky left-0 z-30 bg-background min-w-[200px]"
                    />
                    {projectGroups.map((group, gIdx) =>
                      group.agents.length > 0 ? (
                        <TableHead
                          key={group.project.id}
                          colSpan={1 + group.agents.length}
                          className={cn(
                            'text-center font-semibold text-sm text-muted-foreground border-b-2 border-primary/30 p-0 bg-muted/40',
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
                    {/* Row header: credential name label + separate kebab for bulk ops */}
                    <TableCell className="sticky left-0 z-10 bg-background font-medium border-r p-0">
                      <div className="flex items-center justify-between gap-1 px-4 py-2 max-w-[280px]">
                        <Tooltip>
                          <TooltipTrigger asChild>
                            <span className="flex items-center gap-1.5 truncate">
                              <KeyRound className="h-3.5 w-3.5 shrink-0 text-muted-foreground" />
                              <span className="truncate">{cred.name}</span>
                            </span>
                          </TooltipTrigger>
                          <TooltipContent side="right">
                            <p>{cred.name}</p>
                          </TooltipContent>
                        </Tooltip>
                        <Popover
                          open={openRowPopovers[cred.id] ?? false}
                          onOpenChange={(open) =>
                            setOpenRowPopovers((prev) => ({ ...prev, [cred.id]: open }))
                          }
                        >
                          <PopoverTrigger asChild>
                            <button
                              type="button"
                              aria-label={`Bulk actions for ${cred.name}`}
                              className="shrink-0 h-6 w-6 flex items-center justify-center rounded-sm text-muted-foreground hover:bg-accent/60 hover:text-foreground transition-colors"
                            >
                              <MoreVertical className="h-3.5 w-3.5" />
                            </button>
                          </PopoverTrigger>
                          <PopoverContent className="w-56 p-2" align="start" side="right">
                            <div className="space-y-1">
                              <p className="text-xs font-medium text-muted-foreground px-2 py-1">
                                {cred.name}
                              </p>
                              <Separator />
                              {onEditCredential && (
                                <Button
                                  variant="ghost"
                                  size="sm"
                                  className="w-full justify-start text-sm"
                                  onClick={() => {
                                    closeRowPopover(cred.id)
                                    onEditCredential(cred)
                                  }}
                                >
                                  <Settings2 className="h-3.5 w-3.5 mr-1.5" />
                                  Manage credential
                                </Button>
                              )}
                              <Separator />
                              {projectGroups.length > 0 && (
                                <Button
                                  variant="ghost"
                                  size="sm"
                                  className="w-full justify-start text-sm"
                                  onClick={() => void bulkBindRowProjects(cred)}
                                >
                                  <ShieldCheck className="h-3.5 w-3.5 mr-1.5" />
                                  Grant to all projects
                                </Button>
                              )}
                              <Button
                                variant="destructive"
                                size="sm"
                                className="w-full justify-start text-sm"
                                onClick={() => bulkUnbindRow(cred)}
                              >
                                <ShieldOff className="h-3.5 w-3.5 mr-1.5" />
                                Revoke all access
                              </Button>
                            </div>
                          </PopoverContent>
                        </Popover>
                      </div>
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

        {/* --- Bulk confirmation dialog --- */}
        <AlertDialog
          open={bulkConfirm.show}
          onOpenChange={(open) => {
            if (!open) setBulkConfirm(INITIAL_BULK_CONFIRM)
          }}
        >
          <AlertDialogContent>
            <AlertDialogHeader className="flex flex-row items-start gap-3">
              <div className={cn(
                'rounded-full p-2 shrink-0 mt-0.5',
                bulkConfirm.variant === 'grant'
                  ? 'bg-emerald-100 dark:bg-emerald-950'
                  : 'bg-red-100 dark:bg-red-950',
              )}>
                {bulkConfirm.variant === 'grant'
                  ? <ShieldCheck className="h-5 w-5 text-emerald-600 dark:text-emerald-400" />
                  : <ShieldOff className="h-5 w-5 text-destructive" />}
              </div>
              <div className="space-y-1">
                <AlertDialogTitle>{bulkConfirm.title}</AlertDialogTitle>
                <AlertDialogDescription>{bulkConfirm.message}</AlertDialogDescription>
              </div>
            </AlertDialogHeader>
            {bulkConfirm.details && (bulkConfirm.details.context.length > 0 || bulkConfirm.details.items.length > 0 || (bulkConfirm.details.groups?.length ?? 0) > 0) && (
              <div className={cn(
                'max-h-[200px] overflow-y-auto rounded-md border-l-4 bg-muted/30 px-3 py-2 text-sm',
                bulkConfirm.variant === 'grant'
                  ? 'border-l-emerald-500'
                  : 'border-l-destructive',
              )}>
                {bulkConfirm.details.context.length > 0 && (
                  <div className="text-muted-foreground mb-2">
                    {bulkConfirm.details.context.map((line, i) => (
                      <div key={i}>{line}</div>
                    ))}
                  </div>
                )}
                <Badge variant="secondary" className="mb-2">
                  {bulkConfirm.count} {bulkConfirm.details?.itemLabel ?? 'item'}{bulkConfirm.count === 1 ? '' : 's'}
                </Badge>

                {/* Flat items */}
                {bulkConfirm.details.items.length > 0 && (
                  <ul className="space-y-0.5">
                    {bulkConfirm.details.items.map((item, i) => (
                      <li key={i} className="flex items-center gap-1.5 text-foreground">
                        {bulkConfirm.variant === 'grant'
                          ? <Plus className="h-3 w-3 shrink-0 text-emerald-600 dark:text-emerald-400" />
                          : <Minus className="h-3 w-3 shrink-0 text-destructive" />}
                        {item}
                      </li>
                    ))}
                  </ul>
                )}

                {/* Grouped items (project → agents) */}
                {bulkConfirm.details.groups && bulkConfirm.details.groups.length > 0 && (
                  <div className="space-y-2">
                    {bulkConfirm.details.groups.map((group, gi) => (
                      <div key={gi}>
                        <div className="flex items-center gap-1.5 font-semibold text-foreground">
                          {bulkConfirm.variant === 'grant'
                            ? <Plus className="h-3 w-3 shrink-0 text-emerald-600 dark:text-emerald-400" />
                            : <Minus className="h-3 w-3 shrink-0 text-destructive" />}
                          {group.label}
                        </div>
                        {group.children.length > 0 && (
                          <ul className="ml-5 mt-0.5 space-y-0.5">
                            {group.children.map((child, ci) => (
                              <li key={ci} className="text-muted-foreground text-xs">
                                └ {child}
                              </li>
                            ))}
                          </ul>
                        )}
                      </div>
                    ))}
                  </div>
                )}
              </div>
            )}
            <AlertDialogFooter>
              <AlertDialogCancel onClick={() => setBulkConfirm(INITIAL_BULK_CONFIRM)}>
                Cancel
              </AlertDialogCancel>
              <AlertDialogAction
                className={bulkConfirm.variant === 'revoke'
                  ? 'bg-destructive text-destructive-foreground hover:bg-destructive/90'
                  : 'bg-emerald-600 text-white hover:bg-emerald-700'}
                onClick={() => {
                  const { onConfirm } = bulkConfirm
                  setBulkConfirm(INITIAL_BULK_CONFIRM)
                  onConfirm()
                }}
              >
                {bulkConfirm.variant === 'grant'
                  ? <ShieldCheck className="h-4 w-4 mr-1.5" />
                  : <ShieldOff className="h-4 w-4 mr-1.5" />}
                {bulkConfirm.confirmLabel}
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
      {/* Project-wide column */}
      <TableHead
        className={cn(
          'text-center min-w-[44px] w-[44px] align-bottom p-0',
          gIdx > 0 && 'border-l-2 border-l-border',
          globalColIndex(projectGroups, gIdx, 0) % 2 === 1 && 'bg-muted/20',
        )}
      >
        <Tooltip>
          <TooltipTrigger asChild>
            <span className="flex items-end justify-center h-full w-full pb-2">
              <span
                className="text-[10px] font-medium text-muted-foreground/50 whitespace-nowrap"
                style={{
                  writingMode: 'vertical-lr',
                  textOrientation: 'mixed',
                }}
              >
                All agents
              </span>
            </span>
          </TooltipTrigger>
          <TooltipContent>
            <p>Project-wide — grants access to all agents in {group.project.name}</p>
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
              role="checkbox"
              aria-checked={projectBound}
              className="h-9 w-9 flex items-center justify-center mx-auto cursor-pointer rounded transition-colors hover:bg-accent/50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-1 disabled:pointer-events-none disabled:opacity-50"
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
                <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
              ) : projectBound ? (
                <span className="h-5 w-5 rounded-sm bg-matrix-bound flex items-center justify-center">
                  <Check className="h-3.5 w-3.5 text-matrix-bound-foreground" />
                </span>
              ) : (
                <span className="h-5 w-5 rounded-sm border-2 border-muted-foreground/30 inline-block" />
              )}
            </button>
          </TooltipTrigger>
          <TooltipContent>
            <p>
              {projectBound ? 'Revoke from' : 'Grant to'} project: {group.project.name}
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
            {inherited && !agentBound ? (
              /* Inherited state: clickable to add direct binding on top */
              <Tooltip>
                <TooltipTrigger asChild>
                  <button
                    type="button"
                    role="checkbox"
                    aria-checked="mixed"
                    className="h-9 w-9 flex items-center justify-center mx-auto cursor-pointer rounded transition-colors hover:bg-accent/50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-1 disabled:pointer-events-none disabled:opacity-50"
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
                      <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
                    ) : (
                      <span className="h-5 w-5 rounded-sm border-2 border-dashed border-matrix-bound/60 bg-matrix-bound/10 flex items-center justify-center">
                        <Link2 className="h-3 w-3 text-matrix-bound" />
                      </span>
                    )}
                  </button>
                </TooltipTrigger>
                <TooltipContent>
                  <p>Inherited from project. Click to add direct binding.</p>
                </TooltipContent>
              </Tooltip>
            ) : (
              /* Direct binding or unbound state */
              <Tooltip>
                <TooltipTrigger asChild>
                  <button
                    type="button"
                    role="checkbox"
                    aria-checked={agentBound}
                    className="h-9 w-9 flex items-center justify-center mx-auto cursor-pointer rounded transition-colors hover:bg-accent/50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-1 disabled:pointer-events-none disabled:opacity-50"
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
                      <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
                    ) : agentBound ? (
                      <span className="h-5 w-5 rounded-sm bg-matrix-bound flex items-center justify-center">
                        <Check className="h-3.5 w-3.5 text-matrix-bound-foreground" />
                      </span>
                    ) : (
                      <span className="h-5 w-5 rounded-sm border-2 border-muted-foreground/30 inline-block" />
                    )}
                  </button>
                </TooltipTrigger>
                <TooltipContent>
                  <p>
                    {agentBound ? 'Revoke from' : 'Grant to'} agent: {agent.displayName ?? agent.name}
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
