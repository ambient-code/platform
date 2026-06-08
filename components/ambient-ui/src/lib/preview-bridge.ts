/**
 * PostMessage bridge utilities for cross-origin iframe element capture.
 *
 * The preview bridge script (public/preview-bridge.js) must be included in
 * the previewed page for these utilities to receive responses.
 */

// -- Message types ----------------------------------------------------------

export type CaptureRequest = { type: 'ambient-capture'; x: number; y: number }

export type CaptureResponse = {
  type: 'ambient-captured'
  html: string | null
  tagName?: string
  id?: string | null
  className?: string | null
  textContent?: string | null
  rect: { x: number; y: number; width: number; height: number } | null
}

export type HoverRequest = { type: 'ambient-hover'; x: number; y: number }

export type HoverClearRequest = { type: 'ambient-hover-clear' }

export type HoverResponse = {
  type: 'ambient-hovered'
  rect: { x: number; y: number; width: number; height: number }
}

export type BridgeMessage =
  | CaptureRequest
  | CaptureResponse
  | HoverRequest
  | HoverClearRequest
  | HoverResponse

// -- Guard -------------------------------------------------------------------

function isCaptureResponse(data: unknown): data is CaptureResponse {
  return (
    typeof data === 'object' &&
    data !== null &&
    (data as Record<string, unknown>).type === 'ambient-captured'
  )
}

export function isHoverResponse(data: unknown): data is HoverResponse {
  return (
    typeof data === 'object' &&
    data !== null &&
    (data as Record<string, unknown>).type === 'ambient-hovered'
  )
}

// -- Bridge timeout ----------------------------------------------------------

const BRIDGE_TIMEOUT_MS = 1000

// -- Public API --------------------------------------------------------------

/**
 * Request element capture at (x, y) inside the iframe via postMessage.
 * Returns a promise that resolves with the captured element info, or
 * with `{ html: null, rect: null }` if the bridge does not respond
 * within the timeout.
 */
export function requestCapture(
  iframe: HTMLIFrameElement,
  x: number,
  y: number
): Promise<CaptureResponse> {
  return new Promise((resolve) => {
    const timeout = setTimeout(() => {
      window.removeEventListener('message', handler)
      resolve({ type: 'ambient-captured', html: null, rect: null })
    }, BRIDGE_TIMEOUT_MS)

    function handler(e: MessageEvent) {
      if (!isCaptureResponse(e.data)) return
      clearTimeout(timeout)
      window.removeEventListener('message', handler)
      resolve(e.data)
    }

    window.addEventListener('message', handler)
    iframe.contentWindow?.postMessage(
      { type: 'ambient-capture', x, y } satisfies CaptureRequest,
      '*'
    )
  })
}

/**
 * Send a hover position to the bridge so it can highlight the element
 * under the cursor inside the iframe.
 */
export function requestHover(
  iframe: HTMLIFrameElement,
  x: number,
  y: number
): void {
  iframe.contentWindow?.postMessage(
    { type: 'ambient-hover', x, y } satisfies HoverRequest,
    '*'
  )
}

/**
 * Tell the bridge to clear any active hover highlight.
 */
export function clearHover(iframe: HTMLIFrameElement): void {
  iframe.contentWindow?.postMessage(
    { type: 'ambient-hover-clear' } satisfies HoverClearRequest,
    '*'
  )
}
