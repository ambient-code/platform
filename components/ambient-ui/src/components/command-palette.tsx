'use client'

import { useEffect, useState, useMemo, useCallback } from 'react'
import { useRouter, useParams } from 'next/navigation'
import { useQueries } from '@tanstack/react-query'
import { Monitor, Bot, KeyRound, Clock, FolderOpen } from 'lucide-react'
import {
  CommandDialog,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
  CommandSeparator,
} from '@/components/ui/command'
import { useProjects } from '@/queries/use-projects'
import { queryKeys } from '@/queries/query-keys'
import { createSessionsAdapter } from '@/adapters/sdk-sessions'
import { createAgentsAdapter } from '@/adapters/sdk-agents'
import { useRecentVisits } from '@/hooks/use-recent-visits'
import type { RecentVisitItem } from '@/hooks/use-recent-visits'
import type { DomainSession, DomainAgent, DomainProject } from '@/domain/types'

const sessionsAdapter = createSessionsAdapter()
const agentsAdapter = createAgentsAdapter()

const DEBOUNCE_MS = 300
const MAX_RECENT_DISPLAY = 10

type ProjectGroup = {
  project: DomainProject
  sessions: DomainSession[]
  agents: DomainAgent[]
}

const ICON_MAP: Record<RecentVisitItem['type'], typeof Monitor> = {
  session: Monitor,
  agent: Bot,
  credential: KeyRound,
  project: FolderOpen,
}

export function CommandPalette() {
  const [open, setOpen] = useState(false)
  const [inputValue, setInputValue] = useState('')
  const [debouncedQuery, setDebouncedQuery] = useState('')
  const router = useRouter()
  const params = useParams<{ projectId?: string }>()
  const currentProjectId = params?.projectId ?? ''

  const { recentItems, recordVisit } = useRecentVisits()

  const isSearching = debouncedQuery.length > 0

  // -----------------------------------------------------------------------
  // Debounce: update debouncedQuery 300ms after the user stops typing
  // -----------------------------------------------------------------------
  useEffect(() => {
    if (inputValue.trim() === '') {
      setDebouncedQuery('')
      return
    }
    const timer = setTimeout(() => {
      setDebouncedQuery(inputValue.trim())
    }, DEBOUNCE_MS)
    return () => clearTimeout(timer)
  }, [inputValue])

  // Reset search when palette closes
  useEffect(() => {
    if (!open) {
      setInputValue('')
      setDebouncedQuery('')
    }
  }, [open])

  // -----------------------------------------------------------------------
  // Fetch projects (lightweight, always available for grouping search results)
  // -----------------------------------------------------------------------
  const { data: projectsData } = useProjects()
  const projects = useMemo(() => projectsData?.items ?? [], [projectsData])

  // -----------------------------------------------------------------------
  // Search queries -- only fire when the user has typed something (debounced)
  // -----------------------------------------------------------------------
  const sessionQueries = useQueries({
    queries: projects.map((p) => ({
      queryKey: queryKeys.sessions.list(p.id, { size: 50, search: debouncedQuery }),
      queryFn: () => sessionsAdapter.list(p.id, { size: 50, search: debouncedQuery }),
      enabled: open && isSearching && projects.length > 0,
      staleTime: 15_000,
    })),
  })

  const agentQueries = useQueries({
    queries: projects.map((p) => ({
      queryKey: queryKeys.agents.list(p.id, { size: 50, search: debouncedQuery }),
      queryFn: () => agentsAdapter.list(p.id, { size: 50, search: debouncedQuery }),
      enabled: open && isSearching && projects.length > 0,
      staleTime: 15_000,
    })),
  })

  const groups = useMemo<ProjectGroup[]>(() => {
    if (!isSearching) return []
    return projects.map((p, i) => ({
      project: p,
      sessions: sessionQueries[i]?.data?.items ?? [],
      agents: agentQueries[i]?.data?.items ?? [],
    }))
  }, [projects, sessionQueries, agentQueries, isSearching])

  // Sort: current project first, then alphabetical
  const sortedGroups = useMemo(() => {
    return [...groups].sort((a, b) => {
      if (a.project.id === currentProjectId) return -1
      if (b.project.id === currentProjectId) return 1
      return a.project.name.localeCompare(b.project.name)
    })
  }, [groups, currentProjectId])

  const hasAnySearchResults = sortedGroups.some(
    (g) => g.sessions.length > 0 || g.agents.length > 0,
  )

  const isLoading =
    isSearching &&
    (sessionQueries.some((q) => q.isLoading) || agentQueries.some((q) => q.isLoading))

  // -----------------------------------------------------------------------
  // Recently visited items (top 10)
  // -----------------------------------------------------------------------
  const displayedRecents = useMemo(
    () => recentItems.slice(0, MAX_RECENT_DISPLAY),
    [recentItems],
  )

  // -----------------------------------------------------------------------
  // Keyboard shortcut + custom event
  // -----------------------------------------------------------------------
  useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) {
      if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
        e.preventDefault()
        setOpen((prev) => !prev)
      }
    }

    function handleOpenPalette() {
      setOpen(true)
    }

    document.addEventListener('keydown', handleKeyDown)
    document.addEventListener('open-command-palette', handleOpenPalette)
    return () => {
      document.removeEventListener('keydown', handleKeyDown)
      document.removeEventListener('open-command-palette', handleOpenPalette)
    }
  }, [])

  // -----------------------------------------------------------------------
  // Selection handlers -- record visit then navigate
  // -----------------------------------------------------------------------
  const handleSelectSession = useCallback(
    (project: DomainProject, session: DomainSession) => {
      const href = `/${project.id}/sessions/${session.id}`
      recordVisit({
        type: 'session',
        id: session.id,
        projectId: project.id,
        label: session.name,
        sublabel: [session.phase, session.agentName].filter(Boolean).join(' · ') || null,
        href,
      })
      setOpen(false)
      router.push(href)
    },
    [recordVisit, router],
  )

  const handleSelectAgent = useCallback(
    (project: DomainProject, agent: DomainAgent) => {
      const href = `/${project.id}/agents/${agent.id}`
      recordVisit({
        type: 'agent',
        id: agent.id,
        projectId: project.id,
        label: agent.displayName ?? agent.name,
        sublabel: agent.displayName ? agent.name : null,
        href,
      })
      setOpen(false)
      router.push(href)
    },
    [recordVisit, router],
  )

  const handleSelectCredentials = useCallback(() => {
    recordVisit({
      type: 'credential',
      id: 'credentials',
      projectId: null,
      label: 'Credentials',
      sublabel: null,
      href: '/credentials',
    })
    setOpen(false)
    router.push('/credentials')
  }, [recordVisit, router])

  const handleSelectRecent = useCallback(
    (item: RecentVisitItem) => {
      recordVisit({
        type: item.type,
        id: item.id,
        projectId: item.projectId,
        label: item.label,
        sublabel: item.sublabel,
        href: item.href,
      })
      setOpen(false)
      router.push(item.href)
    },
    [recordVisit, router],
  )

  // -----------------------------------------------------------------------
  // Render
  // -----------------------------------------------------------------------
  return (
    <CommandDialog
      open={open}
      onOpenChange={setOpen}
      title="Search"
      description="Search across all projects"
      shouldFilter={!isSearching}
    >
      <CommandInput
        placeholder="Search sessions, agents, and more..."
        value={inputValue}
        onValueChange={setInputValue}
      />
      <CommandList>
        {/* ---- Empty state when searching ---- */}
        {isSearching && !isLoading && !hasAnySearchResults && (
          <CommandEmpty>No results found.</CommandEmpty>
        )}

        {/* ---- Empty state when no recent items and not searching ---- */}
        {!isSearching && displayedRecents.length === 0 && (
          <CommandEmpty>
            No recent items. Start typing to search across all projects.
          </CommandEmpty>
        )}

        {/* ---- Loading indicator ---- */}
        {isLoading && (
          <div className="py-6 text-center text-sm text-muted-foreground">
            Searching...
          </div>
        )}

        {/* ---- Recently visited (only when NOT searching) ---- */}
        {!isSearching && displayedRecents.length > 0 && (
          <CommandGroup heading="Recent">
            {displayedRecents.map((item) => {
              const Icon = ICON_MAP[item.type]
              return (
                <CommandItem
                  key={`recent-${item.type}-${item.id}`}
                  value={`recent-${item.type}-${item.label}-${item.id}`}
                  onSelect={() => handleSelectRecent(item)}
                >
                  <Icon className="mr-2 size-4" />
                  <div className="flex flex-1 items-center justify-between">
                    <div className="flex flex-col">
                      <span className="text-sm">{item.label}</span>
                      {item.sublabel && (
                        <span className="text-xs text-muted-foreground">
                          {item.sublabel}
                        </span>
                      )}
                    </div>
                    <Clock className="size-3 text-muted-foreground" />
                  </div>
                </CommandItem>
              )
            })}
          </CommandGroup>
        )}

        {/* ---- Search results grouped by project ---- */}
        {isSearching &&
          sortedGroups.map((group) => {
            const hasSessions = group.sessions.length > 0
            const hasAgents = group.agents.length > 0
            if (!hasSessions && !hasAgents) return null

            return (
              <CommandGroup
                key={group.project.id}
                heading={group.project.name}
              >
                {group.sessions.map((session) => (
                  <CommandItem
                    key={session.id}
                    value={`session-${group.project.name}-${session.name}-${session.id}`}
                    onSelect={() => handleSelectSession(group.project, session)}
                  >
                    <Monitor className="mr-2 size-4" />
                    <div className="flex flex-col">
                      <span className="text-sm">{session.name}</span>
                      <span className="text-xs text-muted-foreground">
                        {session.phase}
                        {session.agentName ? ` · ${session.agentName}` : ''}
                      </span>
                    </div>
                  </CommandItem>
                ))}
                {hasSessions && hasAgents && <CommandSeparator />}
                {group.agents.map((agent) => (
                  <CommandItem
                    key={agent.id}
                    value={`agent-${group.project.name}-${agent.displayName ?? agent.name}-${agent.id}`}
                    onSelect={() => handleSelectAgent(group.project, agent)}
                  >
                    <Bot className="mr-2 size-4" />
                    <div className="flex flex-col">
                      <span className="text-sm">
                        {agent.displayName ?? agent.name}
                      </span>
                      {agent.displayName && (
                        <span className="text-xs text-muted-foreground">
                          {agent.name}
                        </span>
                      )}
                    </div>
                  </CommandItem>
                ))}
              </CommandGroup>
            )
          })}

        {/* ---- Quick navigation (below recents or search results) ---- */}
        <CommandGroup heading="Navigate">
          <CommandItem
            value="go-credentials"
            onSelect={handleSelectCredentials}
          >
            <KeyRound className="mr-2 size-4" />
            <span className="text-sm">Credentials</span>
          </CommandItem>
        </CommandGroup>
      </CommandList>
    </CommandDialog>
  )
}
