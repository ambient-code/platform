'use client'

import { useRef, useEffect, useState, useCallback, useMemo } from 'react'
import { useRouter, useParams } from 'next/navigation'
import { X, Plus, PanelRightClose, PanelRightOpen, GripVertical, ChevronUp, ChevronDown, RotateCcw, Save, Trash2 } from 'lucide-react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip'
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
import { useChatSidebar, MAX_SESSIONS, type SidebarSession } from '@/components/chat-sidebar-context'
import {
  ChatItemsList,
  ChatInput,
  buildChatItems,
  PhaseIndicator,
} from '@/components/chat-messages'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { useSession, useStopSession, useDeleteSession, useCreateSession } from '@/queries/use-sessions'
import { useAgents } from '@/queries/use-agents'
import { useSessionMessages } from '@/queries/use-session-messages'
import { useLiveTail, LiveIndicator } from '@/app/(dashboard)/[projectId]/sessions/[sessionId]/_components/live-tail-indicator'
import { getPhaseStyle } from '@/lib/status-colors'
import type { SessionPhase } from '@/domain/types'

const MIN_WIDTH = 320
const MAX_WIDTH = 800
const DEFAULT_WIDTH = 420
const COLLAPSED_WIDTH = 40

const RUNNING_PHASES: ReadonlySet<SessionPhase> = new Set(['Running', 'Pending', 'Creating'])

/** Small colored dot representing session phase */
function PhaseDot({ phase, active, size = 'sm' }: { phase: string; active?: boolean; size?: 'sm' | 'md' }) {
  const style = getPhaseStyle(phase as SessionPhase)
  const variantColors: Record<string, string> = {
    success: 'bg-green-500',
    error: 'bg-red-500',
    warning: 'bg-amber-500',
    info: 'bg-blue-500',
    default: 'bg-muted-foreground',
  }
  const color = variantColors[style.variant] ?? variantColors.default
  const sz = size === 'md' ? 'h-3 w-3' : 'h-2 w-2'
  return (
    <span
      className={`inline-block rounded-full ${sz} ${color} ${active ? 'ring-2 ring-primary ring-offset-1 ring-offset-background' : ''}`}
      aria-label={`Phase: ${phase}`}
    />
  )
}

function TabStrip({
  sessions,
  activeId,
  onSwitch,
  onClose,
  getPhase,
  projectId,
}: {
  sessions: SidebarSession[]
  activeId: string | null
  onSwitch: (id: string) => void
  onClose: (id: string) => void
  getPhase: (id: string) => string
  projectId: string
}) {
  return (
    <div
      className="flex border-b min-h-[32px] overflow-x-auto scrollbar-none"
      role="tablist"
      aria-label="Chat session tabs"
    >
      {sessions.map(s => {
        const isActive = s.sessionId === activeId
        return (
          <div
            key={s.sessionId}
            role="tab"
            tabIndex={0}
            aria-selected={isActive}
            className={`group flex items-center gap-1.5 px-2.5 py-1 text-xs min-w-0 flex-1 truncate transition-colors cursor-pointer ${
              isActive
                ? 'border-b-2 border-primary text-foreground'
                : 'text-muted-foreground hover:text-foreground hover:bg-muted/50'
            }`}
            onClick={() => onSwitch(s.sessionId)}
            onKeyDown={e => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); onSwitch(s.sessionId) } }}
          >
            <PhaseDot phase={getPhase(s.sessionId)} />
            <span className="truncate">{s.sessionName ?? s.agentName ?? s.sessionId.slice(0, 8)}</span>
            <button
              type="button"
              className="ml-auto shrink-0 opacity-0 group-hover:opacity-100 hover:text-destructive transition-opacity"
              onClick={e => { e.stopPropagation(); onClose(s.sessionId) }}
              aria-label="Close tab"
            >
              <X className="h-3 w-3" />
            </button>
          </div>
        )
      })}
      {projectId && <NewSessionButton projectId={projectId} />}
    </div>
  )
}

/** Test-mode toolbar with re-run, save, delete */
function TestToolbar({ session, activeSession, projectId }: { session: { phase: string; id: string }; activeSession: SidebarSession; projectId: string }) {
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false)
  const { closeSession, openTestSidebar } = useChatSidebar()
  const stopSession = useStopSession()
  const deleteSession = useDeleteSession()
  const createSession = useCreateSession()

  const phase = session.phase as SessionPhase
  const isActive = RUNNING_PHASES.has(phase)

  const handleRerun = useCallback(() => {
    const doCreate = () => {
      const newName = `test-${activeSession.agentName ?? 'agent'}-${Date.now()}`
      createSession.mutate(
        {
          name: newName,
          projectId,
          agentId: activeSession.agentId,
          prompt: activeSession.agentPrompt ?? undefined,
          model: activeSession.agentModel ?? undefined,
          annotations: { 'ambient-code.io/ui/test-session': 'true' },
        },
        {
          onSuccess: (s) => {
            closeSession(activeSession.sessionId)
            openTestSidebar({
              sessionId: s.id,
              agentId: activeSession.agentId ?? '',
              agentName: activeSession.agentName ?? '',
              agentPrompt: activeSession.agentPrompt ?? null,
              agentModel: activeSession.agentModel ?? null,
            })
          },
          onError: () => {
            console.error('[TestToolbar] Re-run failed')
          },
        },
      )
    }
    if (isActive) {
      stopSession.mutate(activeSession.sessionId, { onSettled: doCreate })
    } else {
      doCreate()
    }
  }, [activeSession, isActive, stopSession, createSession, closeSession, openTestSidebar])

  const handleSave = useCallback(() => {
    closeSession(activeSession.sessionId)
  }, [closeSession, activeSession.sessionId])

  const handleDelete = useCallback(() => {
    const doDelete = () => {
      deleteSession.mutate(activeSession.sessionId, {
        onSuccess: () => {
          setDeleteDialogOpen(false)
          closeSession(activeSession.sessionId)
        },
      })
    }
    if (isActive) {
      stopSession.mutate(activeSession.sessionId, { onSettled: doDelete })
    } else {
      doDelete()
    }
  }, [activeSession.sessionId, isActive, stopSession, deleteSession, closeSession])

  return (
    <>
      <div className="flex items-center gap-1 border-b px-3 py-1 shrink-0">
        <Button variant="ghost" size="sm" className="h-7 text-xs" onClick={handleRerun} disabled={stopSession.isPending}>
          <RotateCcw className="size-3.5 mr-1" /> Re-run
        </Button>
        <Button variant="ghost" size="sm" className="h-7 text-xs" onClick={handleSave}>
          <Save className="size-3.5 mr-1" /> Save
        </Button>
        <Button variant="ghost" size="sm" className="h-7 text-xs text-destructive hover:text-destructive" onClick={() => setDeleteDialogOpen(true)} disabled={deleteSession.isPending || stopSession.isPending}>
          <Trash2 className="size-3.5 mr-1" /> Delete
        </Button>
      </div>

      <AlertDialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete test session?</AlertDialogTitle>
            <AlertDialogDescription>This will stop and permanently delete this test session.</AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction onClick={handleDelete} className="bg-destructive text-white hover:bg-destructive/90">Delete</AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  )
}

function NewSessionButton({ projectId }: { projectId: string }) {
  const [open, setOpen] = useState(false)
  const { data: agentsData } = useAgents(projectId)
  const createSession = useCreateSession()
  const { openSidebar, canAddSession } = useChatSidebar()
  const agents = agentsData?.items ?? []

  const handleSelect = useCallback((agent: { id: string; name: string; displayName: string | null }) => {
    if (!canAddSession()) {
      toast.warning(`Maximum of ${MAX_SESSIONS} sidebar sessions reached. Close one first.`)
      setOpen(false)
      return
    }
    const sessionName = `${agent.displayName ?? agent.name}-${Date.now()}`
    createSession.mutate(
      { name: sessionName, projectId, agentId: agent.id },
      {
        onSuccess: (session) => {
          openSidebar(session.id, session.name)
          setOpen(false)
        },
      },
    )
  }, [createSession, projectId, openSidebar, canAddSession])

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <button
          type="button"
          className="shrink-0 px-2 py-1 text-muted-foreground hover:text-foreground hover:bg-muted/50 transition-colors"
          aria-label="New session"
        >
          <Plus className="h-3.5 w-3.5" />
        </button>
      </PopoverTrigger>
      <PopoverContent className="w-56 p-1" align="end" side="bottom">
        <div className="px-2 py-1.5 text-xs font-medium text-muted-foreground">
          Select an agent
        </div>
        {agents.length === 0 && (
          <div className="px-2 py-3 text-xs text-muted-foreground text-center">No agents found</div>
        )}
        {agents.map((agent) => (
          <button
            key={agent.id}
            type="button"
            className="w-full flex items-center gap-2 rounded-sm px-2 py-1.5 text-sm hover:bg-muted transition-colors text-left"
            onClick={() => handleSelect(agent)}
            disabled={createSession.isPending}
          >
            <span className="truncate">{agent.displayName ?? agent.name}</span>
            {agent.model && (
              <span className="ml-auto text-[10px] text-muted-foreground shrink-0">{agent.model}</span>
            )}
          </button>
        ))}
      </PopoverContent>
    </Popover>
  )
}

export function ChatSidebar() {
  const { sessions, activeSessionId, isOpen, closeSidebar, closeSession, switchSession } = useChatSidebar()
  const router = useRouter()
  const params = useParams<{ projectId?: string }>()
  const projectId = params?.projectId ?? ''
  const [collapsed, setCollapsed] = useState(false)
  const [width, setWidth] = useState(DEFAULT_WIDTH)
  const [showScrollToTop, setShowScrollToTop] = useState(false)
  const isDragging = useRef(false)
  const startX = useRef(0)
  const startWidth = useRef(DEFAULT_WIDTH)

  const activeSession = sessions.find(s => s.sessionId === activeSessionId) ?? null

  const { data: session } = useSession(activeSessionId ?? '', undefined)
  const { data: messagesData, isLoading: messagesLoading } = useSessionMessages(activeSessionId ?? '')

  const chatItems = useMemo(() => buildChatItems(messagesData?.items ?? []), [messagesData])

  const { scrollRef, sentinelRef, isAtBottom, newEventCount, scrollToBottom } = useLiveTail(chatItems.length)

  const [scrolledUp, setScrolledUp] = useState(false)

  // Cache phase per session for tab dots
  const sessionPhases = useRef<Map<string, string>>(new Map())
  if (activeSessionId && session?.phase) {
    sessionPhases.current.set(activeSessionId, session.phase)
  }
  const getPhase = useCallback((id: string) => sessionPhases.current.get(id) ?? 'Pending', [])

  const handleScroll = useCallback(() => {
    const container = scrollRef.current
    if (!container) return
    setShowScrollToTop(container.scrollTop > 300)
    const atBottom = container.scrollHeight - container.scrollTop - container.clientHeight < 50
    setScrolledUp(!atBottom)
  }, [scrollRef])

  const scrollToTop = useCallback(() => {
    scrollRef.current?.scrollTo({ top: 0, behavior: 'smooth' })
  }, [scrollRef])

  // Auto-scroll on initial load
  const hasScrolledOnLoad = useRef(false)
  useEffect(() => {
    if (!messagesLoading && chatItems.length > 0 && !hasScrolledOnLoad.current) {
      hasScrolledOnLoad.current = true
      requestAnimationFrame(() => { scrollToBottom() })
    }
  }, [messagesLoading, chatItems.length, scrollToBottom])

  // Reset scroll tracking when session changes
  useEffect(() => { hasScrolledOnLoad.current = false }, [activeSessionId])

  // Drag-to-resize
  const dragListenersRef = useRef<{ move: (e: MouseEvent) => void; up: () => void } | null>(null)

  const cleanupDrag = useCallback(() => {
    if (dragListenersRef.current) {
      document.removeEventListener('mousemove', dragListenersRef.current.move)
      document.removeEventListener('mouseup', dragListenersRef.current.up)
      dragListenersRef.current = null
    }
    isDragging.current = false
    document.body.style.cursor = ''
    document.body.style.userSelect = ''
  }, [])

  useEffect(() => cleanupDrag, [cleanupDrag])

  const handleMouseDown = useCallback((e: React.MouseEvent) => {
    e.preventDefault()
    cleanupDrag()
    isDragging.current = true
    startX.current = e.clientX
    startWidth.current = width
    document.body.style.cursor = 'col-resize'
    document.body.style.userSelect = 'none'

    const handleMouseMove = (ev: MouseEvent) => {
      if (!isDragging.current) return
      const delta = startX.current - ev.clientX
      setWidth(Math.min(MAX_WIDTH, Math.max(MIN_WIDTH, startWidth.current + delta)))
    }
    const handleMouseUp = () => cleanupDrag()

    dragListenersRef.current = { move: handleMouseMove, up: handleMouseUp }
    document.addEventListener('mousemove', handleMouseMove)
    document.addEventListener('mouseup', handleMouseUp)
  }, [width, cleanupDrag])

  // Escape to close
  useEffect(() => {
    if (!isOpen) return
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key !== 'Escape') return
      const active = document.activeElement
      if (active instanceof HTMLInputElement || active instanceof HTMLTextAreaElement) return
      if (document.querySelector('[role="dialog"]')) return
      closeSidebar()
    }
    document.addEventListener('keydown', handleKeyDown)
    return () => document.removeEventListener('keydown', handleKeyDown)
  }, [isOpen, closeSidebar])

  if (!isOpen || !activeSessionId) return null

  const sessionName = session?.name ?? 'Loading...'
  const sessionPhase = session?.phase ?? 'Pending'

  if (collapsed) {
    return (
      <aside
        className="flex-shrink-0 border-l bg-background flex flex-col items-center py-3 gap-2 h-screen sticky top-0"
        style={{ width: COLLAPSED_WIDTH }}
        aria-label="Chat sidebar (collapsed)"
      >
        <Button variant="ghost" size="icon" className="h-8 w-8" onClick={() => setCollapsed(false)} aria-label="Expand chat sidebar">
          <PanelRightOpen className="h-4 w-4" />
        </Button>
        {sessions.length > 1 ? (
          <div className="flex-1 flex flex-col items-center gap-2 py-2">
            {sessions.map(s => (
              <TooltipProvider key={s.sessionId}>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <button
                      type="button"
                      className="p-1"
                      onClick={() => { switchSession(s.sessionId); setCollapsed(false) }}
                      aria-label={`Switch to session ${s.sessionName ?? s.agentName ?? s.sessionId.slice(0, 8)}`}
                    >
                      <PhaseDot phase={getPhase(s.sessionId)} active={s.sessionId === activeSessionId} size="md" />
                    </button>
                  </TooltipTrigger>
                  <TooltipContent side="left">{s.sessionName ?? s.agentName ?? s.sessionId.slice(0, 8)}</TooltipContent>
                </Tooltip>
              </TooltipProvider>
            ))}
          </div>
        ) : (
          <div className="flex-1 flex items-center">
            <span className="text-xs font-medium text-muted-foreground [writing-mode:vertical-lr] rotate-180 truncate max-h-48" title={sessionName}>
              {sessionName}
            </span>
          </div>
        )}
      </aside>
    )
  }

  return (
    <aside
      className="flex-shrink-0 border-l bg-background flex flex-col relative h-screen sticky top-0 z-20 max-md:!w-screen max-md:fixed max-md:inset-0 max-md:border-l-0"
      style={{ width }}
      aria-label={`Chat sidebar for session ${sessionName}`}
      role="complementary"
    >
      {/* Drag handle */}
      <div
        className="absolute inset-y-0 left-0 w-1 cursor-col-resize hover:bg-primary/20 active:bg-primary/30 z-10 flex items-center"
        onMouseDown={handleMouseDown}
        role="separator"
        aria-orientation="vertical"
        aria-label="Resize chat sidebar"
        tabIndex={0}
        onKeyDown={e => {
          if (e.key === 'ArrowLeft') setWidth(w => Math.min(MAX_WIDTH, w + 20))
          else if (e.key === 'ArrowRight') setWidth(w => Math.max(MIN_WIDTH, w - 20))
        }}
      >
        <GripVertical className="h-4 w-4 text-muted-foreground/50 -ml-1.5 pointer-events-none" aria-hidden="true" />
      </div>

      {/* Tab strip */}
      <TabStrip sessions={sessions} activeId={activeSessionId} onSwitch={switchSession} onClose={closeSession} getPhase={getPhase} projectId={projectId} />

      {/* Header */}
      <div className="flex items-center gap-2 border-b px-3 py-2 min-h-[48px]">
        <button
          type="button"
          className="flex-1 min-w-0 text-left cursor-pointer hover:opacity-80 transition-opacity"
          onClick={() => {
            const projectId = session?.projectId
            if (projectId && activeSessionId) {
              router.push(`/${projectId}/sessions/${activeSessionId}`)
            }
          }}
          title="Go to session detail"
        >
          <div className="flex items-center gap-2">
            <span className="text-sm font-semibold truncate" title={sessionName}>{sessionName}</span>
            <PhaseIndicator phase={sessionPhase} />
          </div>
        </button>
        <TooltipProvider delayDuration={300}>
          <Tooltip>
            <TooltipTrigger asChild>
              <Button variant="ghost" size="icon" className="h-7 w-7" onClick={() => setCollapsed(true)} aria-label="Collapse sidebar">
                <PanelRightClose className="h-4 w-4" />
              </Button>
            </TooltipTrigger>
            <TooltipContent>Collapse sidebar</TooltipContent>
          </Tooltip>
        </TooltipProvider>
      </div>

      {/* Test mode toolbar */}
      {activeSession?.mode === 'test' && session && (
        <TestToolbar session={{ phase: sessionPhase, id: activeSessionId }} activeSession={activeSession} projectId={projectId} />
      )}

      {/* Message area */}
      <div className="flex-1 min-h-0 overflow-hidden relative">
        {isAtBottom && chatItems.length > 0 && (
          <div className="absolute top-2 right-3 z-10"><LiveIndicator /></div>
        )}
        <div ref={scrollRef} className="h-full overflow-y-auto" role="log" aria-label="Chat messages" onScroll={handleScroll}>
          <ChatItemsList items={chatItems} isLoading={messagesLoading} />
          <div ref={sentinelRef} className="h-1" aria-hidden="true" />
        </div>

        {/* Scroll buttons */}
        <TooltipProvider>
          <div className="absolute bottom-3 right-3 z-10 flex flex-col gap-1">
            <div className={`transition-all duration-200 ${showScrollToTop ? 'opacity-100 scale-100' : 'opacity-0 scale-75 pointer-events-none'}`}>
              <Tooltip>
                <TooltipTrigger asChild>
                  <Button variant="outline" size="icon" className="h-7 w-7 rounded-full shadow-md cursor-pointer" onClick={scrollToTop} aria-label="Scroll to top">
                    <ChevronUp className="h-4 w-4" />
                  </Button>
                </TooltipTrigger>
                <TooltipContent side="left">Scroll to top</TooltipContent>
              </Tooltip>
            </div>
            <div className={`transition-all duration-200 ${scrolledUp && chatItems.length > 0 ? 'opacity-100 scale-100' : 'opacity-0 scale-75 pointer-events-none'}`}>
              <Tooltip>
                <TooltipTrigger asChild>
                  <Button variant="outline" size="icon" className="h-7 w-7 rounded-full shadow-md cursor-pointer" onClick={scrollToBottom} aria-label="Scroll to bottom">
                    <ChevronDown className="h-4 w-4" />
                    {newEventCount > 0 && (
                      <span className="absolute -top-1 -right-1 flex h-4 min-w-4 items-center justify-center rounded-full bg-primary text-[10px] font-bold text-primary-foreground px-1">
                        {newEventCount}
                      </span>
                    )}
                  </Button>
                </TooltipTrigger>
                <TooltipContent side="left">Scroll to bottom</TooltipContent>
              </Tooltip>
            </div>
          </div>
        </TooltipProvider>
      </div>

      {/* Input area */}
      <ChatInput sessionId={activeSessionId} phase={sessionPhase} disabled={messagesLoading} />

    </aside>
  )
}
