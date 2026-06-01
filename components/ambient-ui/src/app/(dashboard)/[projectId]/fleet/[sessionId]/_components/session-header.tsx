'use client'

import { useState } from 'react'
import { ExternalLink, Square, RotateCcw } from 'lucide-react'
import type { DomainSession } from '@/domain/types'
import { getPreviewAnnotations } from '@/domain/annotations'
import { useStopSession, useStartSession } from '@/queries/use-sessions'
import { useSendFeedback } from '@/queries/use-send-feedback'
import { PhaseBadge } from '../../_components/phase-badge'
import { formatDuration, formatRelativeTime } from '@/lib/format-timestamp'
import { Button } from '@/components/ui/button'
import { PreviewOverlay } from '@/components/preview/preview-overlay'

const STOPPABLE_PHASES = new Set(['Running', 'Pending', 'Creating'])
const RESTARTABLE_PHASES = new Set(['Completed', 'Failed', 'Stopped'])

export function SessionHeader({ session }: { session: DomainSession }) {
  const [previewOpen, setPreviewOpen] = useState(false)
  const stopSession = useStopSession()
  const startSession = useStartSession()
  const sendFeedback = useSendFeedback()

  const preview = getPreviewAnnotations(session.annotations)
  const canStop = STOPPABLE_PHASES.has(session.phase)
  const canRestart = RESTARTABLE_PHASES.has(session.phase)

  return (
    <>
      <div className="space-y-3">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <h1 className="text-2xl font-semibold">{session.name}</h1>
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
                <ExternalLink />
                Open Preview
              </Button>
            )}

            {canStop && (
              <Button
                variant="destructive"
                size="sm"
                onClick={() => stopSession.mutate(session.id)}
                disabled={stopSession.isPending}
                aria-label="Stop session"
              >
                <Square />
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
                <RotateCcw />
                Restart
              </Button>
            )}
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
