import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import {
  requestCapture,
  requestHover,
  clearHover,
  isHoverResponse,
} from '../preview-bridge'
import type { CaptureResponse, HoverResponse } from '../preview-bridge'

function createMockIframe(): HTMLIFrameElement {
  const postMessage = vi.fn()
  return {
    contentWindow: { postMessage },
  } as unknown as HTMLIFrameElement
}

describe('preview-bridge', () => {
  beforeEach(() => {
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  describe('requestCapture', () => {
    it('sends ambient-capture message to iframe', () => {
      const iframe = createMockIframe()
      // Don't await — just trigger the call
      void requestCapture(iframe, 100, 200)

      expect(iframe.contentWindow?.postMessage).toHaveBeenCalledWith(
        { type: 'ambient-capture', x: 100, y: 200 },
        '*'
      )
    })

    it('resolves with bridge response when received', async () => {
      const iframe = createMockIframe()
      const promise = requestCapture(iframe, 50, 75)

      const response: CaptureResponse = {
        type: 'ambient-captured',
        html: '<button>Click</button>',
        tagName: 'button',
        id: 'submit-btn',
        className: 'btn primary',
        textContent: 'Click',
        rect: { x: 10, y: 20, width: 100, height: 40 },
      }

      // Simulate bridge response via postMessage
      window.dispatchEvent(
        new MessageEvent('message', { data: response })
      )

      const result = await promise
      expect(result).toEqual(response)
    })

    it('resolves with null html/rect on timeout', async () => {
      const iframe = createMockIframe()
      const promise = requestCapture(iframe, 50, 75)

      // Advance past the 1000ms timeout
      vi.advanceTimersByTime(1100)

      const result = await promise
      expect(result).toEqual({
        type: 'ambient-captured',
        html: null,
        rect: null,
      })
    })

    it('ignores non-capture messages', async () => {
      const iframe = createMockIframe()
      const promise = requestCapture(iframe, 50, 75)

      // Send an unrelated message
      window.dispatchEvent(
        new MessageEvent('message', {
          data: { type: 'unrelated', value: 42 },
        })
      )

      // Should not have resolved yet; advance timer to trigger timeout
      vi.advanceTimersByTime(1100)

      const result = await promise
      expect(result.html).toBeNull()
    })

    it('cleans up message listener after response', async () => {
      const removeSpy = vi.spyOn(window, 'removeEventListener')
      const iframe = createMockIframe()
      const promise = requestCapture(iframe, 10, 20)

      const response: CaptureResponse = {
        type: 'ambient-captured',
        html: '<div></div>',
        rect: { x: 0, y: 0, width: 50, height: 50 },
      }

      window.dispatchEvent(
        new MessageEvent('message', { data: response })
      )

      await promise

      expect(removeSpy).toHaveBeenCalledWith(
        'message',
        expect.any(Function)
      )

      removeSpy.mockRestore()
    })

    it('cleans up message listener on timeout', async () => {
      const removeSpy = vi.spyOn(window, 'removeEventListener')
      const iframe = createMockIframe()
      const promise = requestCapture(iframe, 10, 20)

      vi.advanceTimersByTime(1100)
      await promise

      expect(removeSpy).toHaveBeenCalledWith(
        'message',
        expect.any(Function)
      )

      removeSpy.mockRestore()
    })
  })

  describe('requestHover', () => {
    it('sends ambient-hover message to iframe', () => {
      const iframe = createMockIframe()
      requestHover(iframe, 200, 300)

      expect(iframe.contentWindow?.postMessage).toHaveBeenCalledWith(
        { type: 'ambient-hover', x: 200, y: 300 },
        '*'
      )
    })

    it('does not throw when contentWindow is null', () => {
      const iframe = { contentWindow: null } as HTMLIFrameElement
      expect(() => requestHover(iframe, 10, 20)).not.toThrow()
    })
  })

  describe('clearHover', () => {
    it('sends ambient-hover-clear message to iframe', () => {
      const iframe = createMockIframe()
      clearHover(iframe)

      expect(iframe.contentWindow?.postMessage).toHaveBeenCalledWith(
        { type: 'ambient-hover-clear' },
        '*'
      )
    })

    it('does not throw when contentWindow is null', () => {
      const iframe = { contentWindow: null } as HTMLIFrameElement
      expect(() => clearHover(iframe)).not.toThrow()
    })
  })

  describe('isHoverResponse', () => {
    it('returns true for valid hover response', () => {
      const msg: HoverResponse = {
        type: 'ambient-hovered',
        rect: { x: 10, y: 20, width: 100, height: 50 },
      }
      expect(isHoverResponse(msg)).toBe(true)
    })

    it('returns false for null', () => {
      expect(isHoverResponse(null)).toBe(false)
    })

    it('returns false for undefined', () => {
      expect(isHoverResponse(undefined)).toBe(false)
    })

    it('returns false for non-object', () => {
      expect(isHoverResponse('ambient-hovered')).toBe(false)
    })

    it('returns false for wrong type field', () => {
      expect(isHoverResponse({ type: 'ambient-captured' })).toBe(false)
    })

    it('returns false for capture response', () => {
      const msg: CaptureResponse = {
        type: 'ambient-captured',
        html: '<div></div>',
        rect: { x: 0, y: 0, width: 10, height: 10 },
      }
      expect(isHoverResponse(msg)).toBe(false)
    })
  })
})
