'use client'

import { useCallback, useEffect, useRef, useState } from 'react'
import { Monitor, Tablet, Smartphone, X, MessageCircle } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { isAllowedPreviewHost } from '@/lib/preview-hosts'
import { useFeedback } from '@/hooks/use-feedback'
import { FeedbackOverlay } from '@/components/preview/feedback-overlay'
import { FeedbackPanel } from '@/components/preview/feedback-panel'
import { CommentCard } from '@/components/preview/comment-card'
import type { SessionPhase, FeedbackItem, FeedbackBatch } from '@/domain/types'

type DeviceSize = 'desktop' | 'tablet' | 'mobile'

const DEVICE_WIDTHS: Record<DeviceSize, string> = {
  desktop: '100%',
  tablet: '768px',
  mobile: '375px',
}

const DEVICE_LABELS: Record<DeviceSize, string> = {
  desktop: 'Desktop',
  tablet: 'Tablet',
  mobile: 'Mobile',
}

const SANDBOX_POLICY = 'allow-scripts allow-same-origin allow-forms'

function buildProxyUrl(targetUrl: string): string {
  return `/api/preview-proxy?url=${encodeURIComponent(targetUrl)}`
}

type PendingSelection = {
  type: 'element' | 'region'
  position: { x: number; y: number }
  dimensions?: { width: number; height: number }
  capturedHtml?: string
}

export type PreviewOverlayProps = {
  url: string
  title?: string
  sessionId: string
  sessionPhase: SessionPhase
  onClose: () => void
  onSendFeedback?: (batch: FeedbackBatch) => void
}

export function PreviewOverlay({
  url,
  title,
  sessionId,
  sessionPhase,
  onClose,
  onSendFeedback,
}: PreviewOverlayProps) {
  const [deviceSize, setDeviceSize] = useState<DeviceSize>('desktop')
  const [pendingSelection, setPendingSelection] =
    useState<PendingSelection | null>(null)
  const iframeRef = useRef<HTMLIFrameElement>(null)
  const allowed = isAllowedPreviewHost(url)

  const feedback = useFeedback()
  const [bridgeAvailable, setBridgeAvailable] = useState<boolean | null>(null)
  const [iframeBlocked, setIframeBlocked] = useState(false)

  const handleIframeLoad = useCallback(() => {
    const iframe = iframeRef.current
    if (!iframe) return
    try {
      // If we can access contentDocument and it has no body content,
      // the load was likely blocked (X-Frame-Options / CSP frame-ancestors)
      const doc = iframe.contentDocument
      if (doc && doc.body && doc.body.children.length === 0 && !doc.body.textContent?.trim()) {
        setIframeBlocked(true)
      }
    } catch {
      // Cross-origin — can't inspect, but the iframe may have loaded fine.
      // Use a heuristic: if contentWindow exists but we can't access location,
      // the frame loaded something. If contentWindow is null, it was blocked.
      if (!iframe.contentWindow) {
        setIframeBlocked(true)
      }
    }
  }, [])

  const handleKeyDown = useCallback(
    (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        if (pendingSelection) {
          setPendingSelection(null)
          return
        }
        if (feedback.feedbackMode) {
          feedback.exitFeedbackMode()
          return
        }
        onClose()
      }
      if (e.key === 'c' && !feedback.feedbackMode && !pendingSelection) {
        // Don't activate if user is typing in an input
        const target = e.target as HTMLElement
        if (
          target.tagName === 'INPUT' ||
          target.tagName === 'TEXTAREA' ||
          target.isContentEditable
        ) {
          return
        }
        feedback.enterFeedbackMode()
      }
    },
    [onClose, feedback.feedbackMode, feedback.exitFeedbackMode, feedback.enterFeedbackMode, pendingSelection]
  )

  useEffect(() => {
    document.addEventListener('keydown', handleKeyDown)
    // Prevent body scroll while overlay is open
    const prevOverflow = document.body.style.overflow
    document.body.style.overflow = 'hidden'

    return () => {
      document.removeEventListener('keydown', handleKeyDown)
      document.body.style.overflow = prevOverflow
    }
  }, [handleKeyDown])

  const handleBridgeStatus = useCallback((available: boolean) => {
    setBridgeAvailable((prev) => {
      // Once known, don't downgrade from true to false on subsequent calls
      if (prev === true) return true
      return available
    })
  }, [])

  const handleSelect = useCallback(
    (result: {
      type: 'element' | 'region'
      position: { x: number; y: number }
      dimensions?: { width: number; height: number }
      capturedHtml?: string
    }) => {
      setPendingSelection(result)
    },
    []
  )

  const handleCommentSubmit = useCallback(
    (comment: string) => {
      if (!pendingSelection) return

      const iframeEl = iframeRef.current
      const item: FeedbackItem = {
        id: crypto.randomUUID(),
        type: pendingSelection.type,
        comment,
        position: pendingSelection.position,
        dimensions: pendingSelection.dimensions,
        capturedHtml: pendingSelection.capturedHtml,
        viewportWidth: iframeEl?.clientWidth ?? 0,
        viewportHeight: iframeEl?.clientHeight ?? 0,
        deviceSize,
        timestamp: new Date().toISOString(),
      }

      feedback.addItem(item)
      setPendingSelection(null)
    },
    [pendingSelection, deviceSize, feedback]
  )

  const handleSendAll = useCallback(() => {
    if (onSendFeedback && feedback.pendingItems.length > 0) {
      onSendFeedback({
        items: feedback.pendingItems,
        sessionId,
        previewUrl: url,
      })
      feedback.markAsSent()
    }
  }, [onSendFeedback, feedback.pendingItems, feedback.markAsSent, sessionId, url])

  const displayTitle = title ?? `Preview - ${sessionId}`

  const hasFeedbackActivity =
    feedback.pendingItems.length > 0 || feedback.sentItems.length > 0
  const showPanel = feedback.feedbackMode || hasFeedbackActivity

  return (
    <div
      className="fixed inset-0 z-50 flex flex-col bg-black/80"
      role="dialog"
      aria-modal="true"
      aria-label={`Live preview: ${displayTitle}`}
    >
      {/* Header bar */}
      <div className="flex items-center justify-between border-b border-border bg-background px-4 py-2">
        <div className="flex items-center gap-3">
          <Button
            variant="ghost"
            size="icon-sm"
            onClick={onClose}
            aria-label="Close preview"
          >
            <X />
          </Button>
          <span className="text-sm font-medium">{displayTitle}</span>
          {sessionPhase === 'Running' && (
            <span className="inline-flex items-center gap-1 text-xs text-green-600">
              <span className="inline-block size-2 animate-pulse rounded-full bg-green-500" />
              Live
            </span>
          )}
        </div>

        <div className="flex items-center gap-1">
          <DeviceToggle
            current={deviceSize}
            device="desktop"
            icon={<Monitor className="size-4" />}
            onSelect={setDeviceSize}
          />
          <DeviceToggle
            current={deviceSize}
            device="tablet"
            icon={<Tablet className="size-4" />}
            onSelect={setDeviceSize}
          />
          <DeviceToggle
            current={deviceSize}
            device="mobile"
            icon={<Smartphone className="size-4" />}
            onSelect={setDeviceSize}
          />
          <div className="mx-2 h-5 w-px bg-border" aria-hidden="true" />
          <Button
            variant={feedback.feedbackMode ? 'secondary' : 'ghost'}
            size="sm"
            onClick={() =>
              feedback.feedbackMode
                ? feedback.exitFeedbackMode()
                : feedback.enterFeedbackMode()
            }
            aria-label={
              feedback.feedbackMode
                ? 'Exit feedback mode'
                : 'Enter feedback mode'
            }
            aria-pressed={feedback.feedbackMode}
          >
            <MessageCircle className="size-4" />
            Comment
            <kbd className="ml-1 rounded border border-border/50 px-1 text-[10px] font-normal text-muted-foreground">C</kbd>
          </Button>
        </div>
      </div>

      {/* Content area with optional panel */}
      <div className="flex flex-1 overflow-hidden">
        {/* Preview area */}
        <div className="flex flex-1 items-center justify-center overflow-hidden p-4">
          {!allowed ? (
            <div className="max-w-md rounded-lg border border-destructive bg-destructive/10 p-6 text-center">
              <p className="text-sm font-medium text-destructive">
                Preview URL is not on the trusted hosts allowlist
              </p>
              <p className="mt-2 text-xs text-muted-foreground">
                The URL{' '}
                <code className="rounded bg-muted px-1">{url}</code> does not
                match any allowed host pattern.
              </p>
            </div>
          ) : (
            <div
              className="relative h-full"
              style={{
                width: DEVICE_WIDTHS[deviceSize],
                maxWidth: '100%',
                transition: 'width 200ms ease-in-out',
              }}
            >
              <iframe
                ref={iframeRef}
                src={buildProxyUrl(url)}
                sandbox={SANDBOX_POLICY}
                referrerPolicy="no-referrer"
                title={displayTitle}
                className="h-full w-full rounded border border-border bg-white"
                onLoad={handleIframeLoad}
              />

              {/* Iframe blocked by X-Frame-Options or CSP */}
              {iframeBlocked && (
                <div className="absolute inset-0 flex items-center justify-center rounded bg-background/95">
                  <div className="max-w-md space-y-3 p-6 text-center">
                    <p className="text-sm font-medium text-foreground">
                      Preview could not be loaded
                    </p>
                    <p className="text-xs text-muted-foreground">
                      The preview proxy could not load this app. The target may be unreachable or require separate authentication.
                    </p>
                    <a
                      href={url}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="inline-flex items-center gap-1.5 rounded-md bg-primary px-3 py-1.5 text-sm font-medium text-primary-foreground hover:bg-primary/90"
                    >
                      Open in new tab
                    </a>
                  </div>
                </div>
              )}

              {/* Bridge info banner for cross-origin iframes */}
              {feedback.feedbackMode && bridgeAvailable === false && (
                <div className="pointer-events-none absolute bottom-3 left-1/2 z-20 -translate-x-1/2 rounded-md border border-amber-500/30 bg-amber-950/80 px-3 py-1.5 text-xs text-amber-200 shadow-md backdrop-blur-sm">
                  For element capture, add{' '}
                  <code className="rounded bg-amber-900/60 px-1 font-mono text-[10px]">
                    preview-bridge.js
                  </code>{' '}
                  to your app.
                </div>
              )}

              {/* Feedback overlay on top of iframe */}
              {feedback.feedbackMode && !pendingSelection && (
                <FeedbackOverlay
                  iframeRef={iframeRef}
                  onSelect={handleSelect}
                  onBridgeStatus={handleBridgeStatus}
                />
              )}

              {/* Comment card anchored to selection */}
              {pendingSelection && (
                <CommentCard
                  type={pendingSelection.type}
                  position={pendingSelection.position}
                  dimensions={pendingSelection.dimensions}
                  capturedHtml={pendingSelection.capturedHtml}
                  onSubmit={handleCommentSubmit}
                  onCancel={() => setPendingSelection(null)}
                />
              )}
            </div>
          )}
        </div>

        {/* Feedback panel on the right */}
        {showPanel && (
          <FeedbackPanel
            pendingItems={feedback.pendingItems}
            sentItems={feedback.sentItems}
            onRemoveItem={feedback.removeItem}
            onUpdateComment={feedback.updateComment}
            onSendAll={handleSendAll}
          />
        )}
      </div>
    </div>
  )
}

function DeviceToggle({
  current,
  device,
  icon,
  onSelect,
}: {
  current: DeviceSize
  device: DeviceSize
  icon: React.ReactNode
  onSelect: (device: DeviceSize) => void
}) {
  return (
    <Button
      variant={current === device ? 'secondary' : 'ghost'}
      size="icon-sm"
      onClick={() => onSelect(device)}
      aria-label={`${DEVICE_LABELS[device]} view`}
      aria-pressed={current === device}
    >
      {icon}
    </Button>
  )
}
