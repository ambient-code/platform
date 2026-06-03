'use client'

import { useState, useCallback } from 'react'
import { Square, Trash2, X } from 'lucide-react'
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
import type { DomainSession, SessionPhase } from '@/domain/types'
import { useStopSession, useDeleteSession } from '@/queries/use-sessions'

const STOPPABLE_PHASES: ReadonlySet<SessionPhase> = new Set(['Running', 'Creating', 'Pending'])

export function BulkActionBar({
  selectedSessions,
  onClearSelection,
}: {
  selectedSessions: DomainSession[]
  onClearSelection: () => void
}) {
  const [stopDialogOpen, setStopDialogOpen] = useState(false)
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false)

  const stopSession = useStopSession()
  const deleteSession = useDeleteSession()

  const stoppableSessions = selectedSessions.filter(s => STOPPABLE_PHASES.has(s.phase))
  const count = selectedSessions.length

  const handleConfirmStop = useCallback(() => {
    const toStop = stoppableSessions
    let remaining = toStop.length
    for (const session of toStop) {
      stopSession.mutate(session.id, {
        onSettled: () => {
          remaining -= 1
          if (remaining === 0) {
            setStopDialogOpen(false)
            onClearSelection()
          }
        },
      })
    }
  }, [stoppableSessions, stopSession, onClearSelection])

  const handleConfirmDelete = useCallback(() => {
    let remaining = selectedSessions.length
    for (const session of selectedSessions) {
      deleteSession.mutate(session.id, {
        onSettled: () => {
          remaining -= 1
          if (remaining === 0) {
            setDeleteDialogOpen(false)
            onClearSelection()
          }
        },
      })
    }
  }, [selectedSessions, deleteSession, onClearSelection])

  if (count === 0) return null

  return (
    <>
      <div className="flex items-center gap-3 rounded-lg border bg-muted/50 px-4 py-2 shadow-sm">
        <span className="text-sm font-medium">
          {count} selected
        </span>

        <div className="flex items-center gap-2 ml-auto">
          {stoppableSessions.length > 0 && (
            <Button
              variant="outline"
              size="sm"
              onClick={() => setStopDialogOpen(true)}
              disabled={stopSession.isPending}
            >
              <Square className="size-3.5 mr-1.5" />
              Stop All ({stoppableSessions.length})
            </Button>
          )}

          <Button
            variant="outline"
            size="sm"
            onClick={() => setDeleteDialogOpen(true)}
            disabled={deleteSession.isPending}
            className="text-destructive hover:text-destructive"
          >
            <Trash2 className="size-3.5 mr-1.5" />
            Delete All ({count})
          </Button>

          <Button
            variant="ghost"
            size="icon"
            className="size-7"
            onClick={onClearSelection}
            aria-label="Clear selection"
          >
            <X className="size-3.5" />
          </Button>
        </div>
      </div>

      {/* Stop confirmation dialog */}
      <AlertDialog open={stopDialogOpen} onOpenChange={setStopDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>
              Stop {stoppableSessions.length} {stoppableSessions.length === 1 ? 'session' : 'sessions'}?
            </AlertDialogTitle>
            <AlertDialogDescription>
              The selected running agents will be terminated. Any in-progress work will be lost.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={handleConfirmStop}
              className="bg-destructive text-white hover:bg-destructive/90"
            >
              Stop {stoppableSessions.length === 1 ? 'session' : 'sessions'}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* Delete confirmation dialog */}
      <AlertDialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>
              Delete {count} {count === 1 ? 'session' : 'sessions'}?
            </AlertDialogTitle>
            <AlertDialogDescription>
              This action cannot be undone. The selected sessions and all their data
              will be permanently deleted.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={handleConfirmDelete}
              className="bg-destructive text-white hover:bg-destructive/90"
            >
              Delete {count === 1 ? 'session' : 'sessions'}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  )
}
