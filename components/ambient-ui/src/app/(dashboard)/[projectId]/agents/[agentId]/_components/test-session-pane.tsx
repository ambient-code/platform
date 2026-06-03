'use client'

import { useState, useEffect, useCallback, useRef } from 'react'
import { X, RotateCcw, Save, Trash2, ChevronDown, ChevronRight, Clock } from 'lucide-react'
import { Button } from '@/components/ui/button'
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
import { PhaseBadge } from '@/app/(dashboard)/[projectId]/sessions/_components/phase-badge'
import {
  useLiveTail,
  LiveIndicator,
  JumpToLatestPill,
} from '@/app/(dashboard)/[projectId]/sessions/[sessionId]/_components/live-tail-indicator'
import {
  ChatItemsList,
  ChatInput,
  buildChatItems,
} from '@/components/chat-messages'
import { useSessionMessages } from '@/queries/use-session-messages'
import { useSession, useStopSession, useDeleteSession, useCreateSession } from '@/queries/use-sessions'
import { formatRelativeTime } from '@/lib/format-timestamp'
import type { SessionPhase } from '@/domain/types'

const MAX_HISTORY = 5

const RUNNING_PHASES: ReadonlySet<SessionPhase> = new Set(['Running', 'Pending', 'Creating'])

type TestHistoryEntry = {
  id: string
  name: string
  phase: SessionPhase
  createdAt: string
}

type TestSessionPaneProps = {
  sessionId: string
  sessionName: string
  projectId: string
  agentId: string
  agentName: string
  agentPrompt: string | null
  agentModel: string | null
  history: TestHistoryEntry[]
  onClose: () => void
  onRunTest: (sessionId: string, name: string) => void
  onSelectHistory: (entry: TestHistoryEntry) => void
}

export type { TestHistoryEntry }

export function TestSessionPane({
  sessionId,
  sessionName,
  projectId,
  agentId,
  agentName,
  agentPrompt,
  agentModel,
  history,
  onClose,
  onRunTest,
  onSelectHistory,
}: TestSessionPaneProps) {
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false)
  const [historyExpanded, setHistoryExpanded] = useState(false)
  const sessionIdRef = useRef(sessionId)

  const { data: session } = useSession(sessionId)
  const { data: messages, isLoading: messagesLoading } = useSessionMessages(sessionId)
  const stopSession = useStopSession()
  const deleteSession = useDeleteSession()
  const createSession = useCreateSession()

  const phase = session?.phase ?? 'Pending'
  const isActive = RUNNING_PHASES.has(phase)

  const chatItems = messages ? buildChatItems(messages.items) : []
  const liveTail = useLiveTail(chatItems.length)

  useEffect(() => {
    sessionIdRef.current = sessionId
  }, [sessionId])

  // Stop running session on browser close/navigation
  useEffect(() => {
    const handleBeforeUnload = () => {
      if (sessionIdRef.current) {
        // Use sendBeacon for reliability during unload
        const url = `/api/sessions/${sessionIdRef.current}/stop`
        navigator.sendBeacon(url)
      }
    }

    window.addEventListener('beforeunload', handleBeforeUnload)
    return () => window.removeEventListener('beforeunload', handleBeforeUnload)
  }, [])

  const handleSave = useCallback(() => {
    // "Save" promotes the test session to a regular session by closing the pane.
    // The session keeps running as a normal session.
    onClose()
  }, [onClose])

  const handleDelete = useCallback(() => {
    if (isActive) {
      stopSession.mutate(sessionId, {
        onSettled: () => {
          deleteSession.mutate(sessionId, {
            onSuccess: () => {
              setDeleteDialogOpen(false)
              onClose()
            },
          })
        },
      })
    } else {
      deleteSession.mutate(sessionId, {
        onSuccess: () => {
          setDeleteDialogOpen(false)
          onClose()
        },
      })
    }
  }, [sessionId, isActive, stopSession, deleteSession, onClose])

  const handleRerun = useCallback(() => {
    const doCreate = () => {
      const newName = `test-${agentName}-${Date.now()}`
      createSession.mutate(
        {
          name: newName,
          projectId,
          agentId,
          prompt: agentPrompt ?? undefined,
          model: agentModel ?? undefined,
          annotations: { 'ambient-code.io/ui/test-session': 'true' },
        },
        { onSuccess: (s) => onRunTest(s.id, s.name) },
      )
    }
    if (isActive) {
      stopSession.mutate(sessionId, { onSettled: doCreate })
    } else {
      doCreate()
    }
  }, [sessionId, isActive, stopSession, createSession, projectId, agentId, agentName, agentPrompt, agentModel, onRunTest])

  const visibleHistory = history.slice(0, MAX_HISTORY)

  return (
    <div className="flex flex-col border-l h-screen sticky top-0">
      {/* Header */}
      <div className="flex items-center justify-between border-b px-3 py-2">
        <div className="flex items-center gap-2 min-w-0">
          <h3 className="text-sm font-medium truncate">{sessionName}</h3>
          <PhaseBadge phase={phase} />
          {isActive && <LiveIndicator />}
        </div>
        <Button
          variant="ghost"
          size="icon"
          className="size-7 shrink-0"
          onClick={onClose}
          aria-label="Close test pane"
        >
          <X className="size-4" />
        </Button>
      </div>

      {/* Toolbar */}
      <div className="flex items-center gap-1 border-b px-3 py-1.5">
        <Button
          variant="ghost"
          size="sm"
          className="h-7 text-xs"
          onClick={handleRerun}
          disabled={stopSession.isPending}
        >
          <RotateCcw className="size-3.5 mr-1" />
          Re-run
        </Button>
        <Button
          variant="ghost"
          size="sm"
          className="h-7 text-xs"
          onClick={handleSave}
        >
          <Save className="size-3.5 mr-1" />
          Save
        </Button>
        <Button
          variant="ghost"
          size="sm"
          className="h-7 text-xs text-destructive hover:text-destructive"
          onClick={() => setDeleteDialogOpen(true)}
          disabled={deleteSession.isPending || stopSession.isPending}
        >
          <Trash2 className="size-3.5 mr-1" />
          Delete
        </Button>
      </div>

      {/* Messages */}
      <div
        ref={liveTail.scrollRef}
        className="flex-1 overflow-y-auto"
      >
        <ChatItemsList items={chatItems} isLoading={messagesLoading} />
        <div ref={liveTail.sentinelRef} className="h-px" />
        <JumpToLatestPill
          newEventCount={liveTail.newEventCount}
          onClick={liveTail.scrollToBottom}
        />
      </div>

      {/* Chat input */}
      <ChatInput sessionId={sessionId} phase={phase} disabled={false} />

      {/* History accordion */}
      {visibleHistory.length > 0 && (
        <div className="border-t">
          <button
            type="button"
            className="flex w-full items-center gap-1.5 px-3 py-2 text-xs font-medium text-muted-foreground hover:text-foreground transition-colors"
            onClick={() => setHistoryExpanded((prev) => !prev)}
            aria-expanded={historyExpanded}
          >
            {historyExpanded ? (
              <ChevronDown className="size-3.5" />
            ) : (
              <ChevronRight className="size-3.5" />
            )}
            Previous runs ({visibleHistory.length})
          </button>
          {historyExpanded && (
            <div className="border-t">
              {visibleHistory.map((entry) => (
                <button
                  key={entry.id}
                  type="button"
                  className="flex w-full items-center gap-2 px-3 py-1.5 text-xs hover:bg-muted/50 transition-colors"
                  onClick={() => onSelectHistory(entry)}
                >
                  <Clock className="size-3 text-muted-foreground shrink-0" />
                  <span className="truncate">{entry.name}</span>
                  <PhaseBadge phase={entry.phase} />
                  <span className="ml-auto text-muted-foreground shrink-0">
                    {formatRelativeTime(entry.createdAt)}
                  </span>
                </button>
              ))}
            </div>
          )}
        </div>
      )}

      {/* Delete confirmation dialog */}
      <AlertDialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete test session?</AlertDialogTitle>
            <AlertDialogDescription>
              This will stop and permanently delete this test session.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={handleDelete}
              className="bg-destructive text-white hover:bg-destructive/90"
            >
              Delete
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
