'use client'

import { useState, useCallback } from 'react'
import { useRouter, useParams } from 'next/navigation'
import { ExternalLink, Square, RotateCcw, Download, Trash2, MoreVertical } from 'lucide-react'
import type { DomainSession } from '@/domain/types'
import { getPreviewAnnotations } from '@/domain/annotations'
import { useStopSession, useStartSession, useDeleteSession } from '@/queries/use-sessions'
import { useSendFeedback } from '@/queries/use-send-feedback'
import { PhaseBadge } from '../../_components/phase-badge'
import { formatDuration, formatRelativeTime } from '@/lib/format-timestamp'
import { Button } from '@/components/ui/button'
import { PreviewOverlay } from '@/components/preview/preview-overlay'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
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

const STOPPABLE_PHASES = new Set(['Running', 'Pending', 'Creating'])
const RESTARTABLE_PHASES = new Set(['Completed', 'Failed', 'Stopped'])

export function SessionHeader({ session }: { session: DomainSession }) {
  const [previewOpen, setPreviewOpen] = useState(false)
  const [stopDialogOpen, setStopDialogOpen] = useState(false)
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false)

  const router = useRouter()
  const { projectId } = useParams<{ projectId: string; sessionId: string }>()

  const stopSession = useStopSession()
  const startSession = useStartSession()
  const deleteSession = useDeleteSession()
  const sendFeedback = useSendFeedback()

  const preview = getPreviewAnnotations(session.annotations)
  const canStop = STOPPABLE_PHASES.has(session.phase)
  const canRestart = RESTARTABLE_PHASES.has(session.phase)

  const handleConfirmStop = useCallback(() => {
    stopSession.mutate(session.id, {
      onSettled: () => setStopDialogOpen(false),
    })
  }, [stopSession, session.id])

  const handleConfirmDelete = useCallback(() => {
    deleteSession.mutate(session.id, {
      onSuccess: () => {
        setDeleteDialogOpen(false)
        router.push(`/${projectId}/sessions`)
      },
      onError: () => setDeleteDialogOpen(false),
    })
  }, [deleteSession, session.id, router, projectId])

  const handleExport = useCallback(async () => {
    const { createSessionMessagesAdapterWithFetch } = await import('@/adapters/session-messages')
    const adapter = createSessionMessagesAdapterWithFetch()
    const result = await adapter.list(session.id, { size: 1000 })

    const exportData = {
      session: {
        id: session.id,
        name: session.name,
        phase: session.phase,
        agentName: session.agentName,
        model: session.model,
        createdAt: session.createdAt,
        startTime: session.startTime,
        completionTime: session.completionTime,
      },
      messages: result.items,
      exportedAt: new Date().toISOString(),
    }

    const blob = new Blob([JSON.stringify(exportData, null, 2)], {
      type: 'application/json',
    })
    const url = URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = `session-${session.name || session.id}.json`
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
    URL.revokeObjectURL(url)
  }, [session])

  return (
    <>
      <div className="sticky top-14 z-[5] bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60 pb-4 -mx-1 px-1">
        <div className="space-y-3">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              <h1 className="text-lg font-semibold">{session.name}</h1>
              <PhaseBadge phase={session.phase} />
            </div>

            <div className="flex items-center gap-2">
              {preview && (
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setPreviewOpen(true)}
                  aria-label="Open preview"
                >
                  <ExternalLink className="size-4" />
                  Preview
                </Button>
              )}

              {canStop && (
                <Button
                  variant="destructive"
                  size="sm"
                  onClick={() => setStopDialogOpen(true)}
                  disabled={stopSession.isPending}
                  aria-label="Stop session"
                >
                  <Square className="size-4" />
                  Stop
                </Button>
              )}

              {canRestart && (
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => startSession.mutate(session.id)}
                  disabled={startSession.isPending}
                  aria-label="Restart session"
                >
                  <RotateCcw className="size-4" />
                  Restart
                </Button>
              )}

              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button variant="ghost" size="icon" className="size-8" aria-label="More actions">
                    <MoreVertical className="size-4" />
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end">
                  <DropdownMenuItem onClick={handleExport}>
                    <Download className="size-4 mr-2" />
                    Export
                  </DropdownMenuItem>
                  <DropdownMenuSeparator />
                  <DropdownMenuItem
                    onClick={() => setDeleteDialogOpen(true)}
                    disabled={deleteSession.isPending}
                    className="text-destructive focus:text-destructive"
                  >
                    <Trash2 className="size-4 mr-2" />
                    Delete
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            </div>
          </div>

          <div className="flex items-center gap-6 text-sm text-muted-foreground">
            {session.agentName && (
              <MetaItem label="Agent" value={session.agentName} />
            )}
            {session.model && (
              <MetaItem label="Model" value={session.model} />
            )}
            {session.startTime && (
              <MetaItem
                label="Duration"
                value={formatDuration(session.startTime, session.completionTime)}
              />
            )}
            <MetaItem label="Created" value={formatRelativeTime(session.createdAt)} />
          </div>
        </div>
      </div>

      {/* Stop confirmation dialog */}
      <AlertDialog open={stopDialogOpen} onOpenChange={setStopDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Stop this session?</AlertDialogTitle>
            <AlertDialogDescription>
              The agent will be terminated. Any in-progress work will be lost.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={handleConfirmStop}
              className="bg-destructive text-white hover:bg-destructive/90"
            >
              Stop session
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* Delete confirmation dialog */}
      <AlertDialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete this session?</AlertDialogTitle>
            <AlertDialogDescription>
              This action cannot be undone. The session and all its data will be
              permanently deleted.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={handleConfirmDelete}
              className="bg-destructive text-white hover:bg-destructive/90"
            >
              Delete session
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {previewOpen && preview && (
        <PreviewOverlay
          url={preview.url}
          title={preview.title}
          sessionId={session.id}
          sessionPhase={session.phase}
          onClose={() => setPreviewOpen(false)}
          onSendFeedback={(batch) => sendFeedback.mutate(batch)}
        />
      )}
    </>
  )
}

function MetaItem({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <span className="text-muted-foreground/70">{label}:</span>{' '}
      <span>{value}</span>
    </div>
  )
}
