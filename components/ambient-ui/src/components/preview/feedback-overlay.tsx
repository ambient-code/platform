'use client'

import { useCallback, useEffect, useRef, useState } from 'react'
import {
  requestCapture,
  requestHover,
  clearHover,
  isHoverResponse,
} from '@/lib/preview-bridge'
import type { HoverResponse } from '@/lib/preview-bridge'

type SelectionRect = {
  startX: number
  startY: number
  width: number
  height: number
}

type HoverHighlight =
  | { kind: 'element'; left: number; top: number; width: number; height: number }
  | { kind: 'crosshair'; x: number; y: number }

type SelectionResult = {
  type: 'element' | 'region'
  position: { x: number; y: number }
  dimensions?: { width: number; height: number }
  capturedHtml?: string
}

export type FeedbackOverlayProps = {
  iframeRef: React.RefObject<HTMLIFrameElement | null>
  onSelect: (result: SelectionResult) => void
  /** Called when a cross-origin capture attempt completes; true if the bridge responded. */
  onBridgeStatus?: (available: boolean) => void
}

const MIN_DRAG_DISTANCE = 5

export function FeedbackOverlay({ iframeRef, onSelect, onBridgeStatus }: FeedbackOverlayProps) {
  const overlayRef = useRef<HTMLDivElement>(null)
  const [isDragging, setIsDragging] = useState(false)
  const [dragStart, setDragStart] = useState<{ x: number; y: number } | null>(
    null
  )
  const [selectionRect, setSelectionRect] = useState<SelectionRect | null>(null)
  const [hoverHighlight, setHoverHighlight] = useState<HoverHighlight | null>(
    null
  )

  const getRelativePosition = useCallback(
    (e: React.MouseEvent) => {
      const overlay = overlayRef.current
      if (!overlay) return { x: 0, y: 0 }
      const rect = overlay.getBoundingClientRect()
      return {
        x: Math.round(e.clientX - rect.left),
        y: Math.round(e.clientY - rect.top),
      }
    },
    []
  )

  /**
   * Try to capture element HTML at (x, y).
   * First attempts same-origin DOM access; falls back to postMessage bridge.
   */
  const tryCaptureDomElement = useCallback(
    async (x: number, y: number): Promise<string | undefined> => {
      if (!iframeRef.current) return undefined

      // Try same-origin access first
      try {
        const doc = iframeRef.current.contentDocument
        if (doc) {
          const el = doc.elementFromPoint(x, y)
          if (el) {
            onBridgeStatus?.(true)
            return el.outerHTML.slice(0, 500)
          }
        }
      } catch {
        // Cross-origin: fall through to postMessage bridge
      }

      // Fall back to postMessage bridge
      const response = await requestCapture(iframeRef.current, x, y)
      const bridgeResponded = response.html !== null
      onBridgeStatus?.(bridgeResponded)
      return response.html ?? undefined
    },
    [iframeRef, onBridgeStatus]
  )

  /**
   * Try to get element bounds at (x, y) for hover highlighting.
   * First attempts same-origin DOM access; for cross-origin, sends a
   * hover request to the bridge (response is handled via message listener).
   */
  const tryGetElementBounds = useCallback(
    (
      x: number,
      y: number
    ): { left: number; top: number; width: number; height: number } | null => {
      const iframe = iframeRef.current
      const overlay = overlayRef.current
      if (!iframe || !overlay) return null

      // Try same-origin access first
      try {
        const doc = iframe.contentDocument
        if (doc) {
          const el = doc.elementFromPoint(x, y)
          if (el) {
            const elRect = el.getBoundingClientRect()
            const iframeRect = iframe.getBoundingClientRect()
            const overlayRect = overlay.getBoundingClientRect()
            return {
              left: Math.round(iframeRect.left - overlayRect.left + elRect.left),
              top: Math.round(iframeRect.top - overlayRect.top + elRect.top),
              width: Math.round(elRect.width),
              height: Math.round(elRect.height),
            }
          }
        }
      } catch {
        // Cross-origin: send hover request to bridge
        requestHover(iframe, x, y)
      }

      return null
    },
    [iframeRef]
  )

  // Listen for hover responses from the bridge (cross-origin case)
  useEffect(() => {
    function handleHoverResponse(e: MessageEvent) {
      if (!isHoverResponse(e.data)) return

      const iframe = iframeRef.current
      const overlay = overlayRef.current
      if (!iframe || !overlay) return

      const hoverData = e.data as HoverResponse
      const iframeRect = iframe.getBoundingClientRect()
      const overlayRect = overlay.getBoundingClientRect()

      setHoverHighlight({
        kind: 'element',
        left: Math.round(iframeRect.left - overlayRect.left + hoverData.rect.x),
        top: Math.round(iframeRect.top - overlayRect.top + hoverData.rect.y),
        width: Math.round(hoverData.rect.width),
        height: Math.round(hoverData.rect.height),
      })
    }

    window.addEventListener('message', handleHoverResponse)
    return () => window.removeEventListener('message', handleHoverResponse)
  }, [iframeRef])

  const handleMouseDown = useCallback(
    (e: React.MouseEvent) => {
      if (e.button !== 0) return
      const pos = getRelativePosition(e)
      setDragStart(pos)
      setIsDragging(false)
      setSelectionRect(null)
    },
    [getRelativePosition]
  )

  const handleMouseMove = useCallback(
    (e: React.MouseEvent) => {
      const pos = getRelativePosition(e)

      if (!dragStart) {
        // Not dragging: update hover highlight
        const bounds = tryGetElementBounds(pos.x, pos.y)
        if (bounds) {
          setHoverHighlight({ kind: 'element', ...bounds })
        } else {
          // Only show crosshair fallback if no bridge response is pending
          // (bridge responses update hoverHighlight via the message listener)
          setHoverHighlight((current) => {
            if (current?.kind === 'element') return current
            return { kind: 'crosshair', ...pos }
          })
        }
        return
      }

      // Dragging: clear hover highlight and handle selection
      setHoverHighlight(null)
      const dx = pos.x - dragStart.x
      const dy = pos.y - dragStart.y

      if (
        !isDragging &&
        Math.sqrt(dx * dx + dy * dy) > MIN_DRAG_DISTANCE
      ) {
        setIsDragging(true)
      }

      if (isDragging || Math.sqrt(dx * dx + dy * dy) > MIN_DRAG_DISTANCE) {
        setSelectionRect({
          startX: Math.min(dragStart.x, pos.x),
          startY: Math.min(dragStart.y, pos.y),
          width: Math.abs(dx),
          height: Math.abs(dy),
        })
      }
    },
    [dragStart, isDragging, getRelativePosition, tryGetElementBounds]
  )

  const handleMouseUp = useCallback(
    async (e: React.MouseEvent) => {
      if (!dragStart) return
      const pos = getRelativePosition(e)

      if (isDragging && selectionRect) {
        // Region selection
        onSelect({
          type: 'region',
          position: {
            x: selectionRect.startX,
            y: selectionRect.startY,
          },
          dimensions: {
            width: selectionRect.width,
            height: selectionRect.height,
          },
        })
      } else {
        // Element click (async: may use postMessage bridge)
        const capturedHtml = await tryCaptureDomElement(pos.x, pos.y)
        onSelect({
          type: 'element',
          position: pos,
          capturedHtml,
        })
      }

      setDragStart(null)
      setIsDragging(false)
      setSelectionRect(null)
    },
    [
      dragStart,
      isDragging,
      selectionRect,
      getRelativePosition,
      onSelect,
      tryCaptureDomElement,
    ]
  )

  const handleMouseLeave = useCallback(() => {
    setHoverHighlight(null)
    // Clear hover highlight inside the iframe (bridge)
    if (iframeRef.current) {
      clearHover(iframeRef.current)
    }
  }, [iframeRef])

  return (
    <div
      ref={overlayRef}
      className="absolute inset-0 z-10"
      style={{ cursor: 'crosshair' }}
      onMouseDown={handleMouseDown}
      onMouseMove={handleMouseMove}
      onMouseUp={handleMouseUp}
      onMouseLeave={handleMouseLeave}
      role="application"
      aria-label="Feedback selection area. Click an element or drag to select a region."
    >
      {/* Instruction bar */}
      <div className="pointer-events-none absolute left-1/2 top-3 -translate-x-1/2 rounded-md bg-background/90 px-4 py-2 text-xs text-muted-foreground shadow-md backdrop-blur-sm">
        Click an element or drag to select a region. Press Esc to cancel.
      </div>

      {/* Hover highlight */}
      {hoverHighlight && !isDragging && hoverHighlight.kind === 'element' && (
        <div
          className="pointer-events-none absolute border-2 border-blue-500 bg-blue-500/5"
          style={{
            left: hoverHighlight.left,
            top: hoverHighlight.top,
            width: hoverHighlight.width,
            height: hoverHighlight.height,
          }}
          aria-hidden="true"
        />
      )}
      {hoverHighlight && !isDragging && hoverHighlight.kind === 'crosshair' && (
        <div
          className="pointer-events-none absolute border-2 border-blue-500 bg-blue-500/5 rounded"
          style={{
            left: hoverHighlight.x - 16,
            top: hoverHighlight.y - 16,
            width: 32,
            height: 32,
          }}
          aria-hidden="true"
        />
      )}

      {/* Selection rectangle while dragging */}
      {selectionRect && (
        <div
          className="pointer-events-none absolute border-2 border-blue-500 bg-blue-500/10"
          style={{
            left: selectionRect.startX,
            top: selectionRect.startY,
            width: selectionRect.width,
            height: selectionRect.height,
          }}
          aria-hidden="true"
        />
      )}
    </div>
  )
}
