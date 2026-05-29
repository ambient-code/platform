import { renderHook, act } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { useFeedback } from '../use-feedback'
import type { FeedbackItem } from '@/domain/types'

function makeFeedbackItem(overrides?: Partial<FeedbackItem>): FeedbackItem {
  return {
    id: crypto.randomUUID(),
    type: 'element',
    comment: 'Test comment',
    position: { x: 100, y: 200 },
    viewportWidth: 1024,
    viewportHeight: 768,
    deviceSize: 'desktop',
    timestamp: new Date().toISOString(),
    ...overrides,
  }
}

describe('useFeedback', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
  })

  describe('feedback mode', () => {
    it('starts with feedback mode disabled', () => {
      const { result } = renderHook(() => useFeedback())
      expect(result.current.feedbackMode).toBe(false)
    })

    it('enters feedback mode', () => {
      const { result } = renderHook(() => useFeedback())
      act(() => result.current.enterFeedbackMode())
      expect(result.current.feedbackMode).toBe(true)
    })

    it('exits feedback mode', () => {
      const { result } = renderHook(() => useFeedback())
      act(() => result.current.enterFeedbackMode())
      act(() => result.current.exitFeedbackMode())
      expect(result.current.feedbackMode).toBe(false)
    })
  })

  describe('adding items', () => {
    it('starts with empty pending and sent arrays', () => {
      const { result } = renderHook(() => useFeedback())
      expect(result.current.pendingItems).toEqual([])
      expect(result.current.sentItems).toEqual([])
    })

    it('adds an item to pending', () => {
      const { result } = renderHook(() => useFeedback())
      const item = makeFeedbackItem()
      act(() => result.current.addItem(item))
      expect(result.current.pendingItems).toHaveLength(1)
      expect(result.current.pendingItems[0]).toEqual(item)
    })

    it('adds multiple items to pending', () => {
      const { result } = renderHook(() => useFeedback())
      const item1 = makeFeedbackItem({ comment: 'First' })
      const item2 = makeFeedbackItem({ comment: 'Second' })
      act(() => {
        result.current.addItem(item1)
        result.current.addItem(item2)
      })
      expect(result.current.pendingItems).toHaveLength(2)
    })

    it('adds a region-type item with dimensions', () => {
      const { result } = renderHook(() => useFeedback())
      const item = makeFeedbackItem({
        type: 'region',
        dimensions: { width: 200, height: 150 },
      })
      act(() => result.current.addItem(item))
      expect(result.current.pendingItems[0].type).toBe('region')
      expect(result.current.pendingItems[0].dimensions).toEqual({
        width: 200,
        height: 150,
      })
    })
  })

  describe('removing items', () => {
    it('removes an item by id', () => {
      const { result } = renderHook(() => useFeedback())
      const item = makeFeedbackItem()
      act(() => result.current.addItem(item))
      expect(result.current.pendingItems).toHaveLength(1)
      act(() => result.current.removeItem(item.id))
      expect(result.current.pendingItems).toHaveLength(0)
    })

    it('does not affect other items when removing', () => {
      const { result } = renderHook(() => useFeedback())
      const item1 = makeFeedbackItem({ comment: 'Keep' })
      const item2 = makeFeedbackItem({ comment: 'Remove' })
      act(() => {
        result.current.addItem(item1)
        result.current.addItem(item2)
      })
      act(() => result.current.removeItem(item2.id))
      expect(result.current.pendingItems).toHaveLength(1)
      expect(result.current.pendingItems[0].comment).toBe('Keep')
    })

    it('is a no-op when removing a non-existent id', () => {
      const { result } = renderHook(() => useFeedback())
      const item = makeFeedbackItem()
      act(() => result.current.addItem(item))
      act(() => result.current.removeItem('non-existent-id'))
      expect(result.current.pendingItems).toHaveLength(1)
    })
  })

  describe('updating comments', () => {
    it('updates the comment of a pending item', () => {
      const { result } = renderHook(() => useFeedback())
      const item = makeFeedbackItem({ comment: 'Original' })
      act(() => result.current.addItem(item))
      act(() => result.current.updateComment(item.id, 'Updated'))
      expect(result.current.pendingItems[0].comment).toBe('Updated')
    })

    it('does not affect other items when updating comment', () => {
      const { result } = renderHook(() => useFeedback())
      const item1 = makeFeedbackItem({ comment: 'Unchanged' })
      const item2 = makeFeedbackItem({ comment: 'Will change' })
      act(() => {
        result.current.addItem(item1)
        result.current.addItem(item2)
      })
      act(() => result.current.updateComment(item2.id, 'Changed'))
      expect(result.current.pendingItems[0].comment).toBe('Unchanged')
      expect(result.current.pendingItems[1].comment).toBe('Changed')
    })
  })

  describe('marking items as sent', () => {
    it('moves all pending items to sent when markAsSent is called', () => {
      const { result } = renderHook(() => useFeedback())
      const item1 = makeFeedbackItem()
      const item2 = makeFeedbackItem()
      act(() => {
        result.current.addItem(item1)
        result.current.addItem(item2)
      })
      act(() => result.current.markAsSent())
      expect(result.current.pendingItems).toHaveLength(0)
      expect(result.current.sentItems).toHaveLength(2)
    })

    it('appends to existing sent items', () => {
      const { result } = renderHook(() => useFeedback())
      const item1 = makeFeedbackItem({ comment: 'Batch 1' })
      act(() => result.current.addItem(item1))
      act(() => result.current.markAsSent())

      const item2 = makeFeedbackItem({ comment: 'Batch 2' })
      act(() => result.current.addItem(item2))
      act(() => result.current.markAsSent())

      expect(result.current.sentItems).toHaveLength(2)
      expect(result.current.pendingItems).toHaveLength(0)
    })
  })

  describe('clearing pending', () => {
    it('clears all pending items', () => {
      const { result } = renderHook(() => useFeedback())
      act(() => {
        result.current.addItem(makeFeedbackItem())
        result.current.addItem(makeFeedbackItem())
      })
      act(() => result.current.clearPending())
      expect(result.current.pendingItems).toHaveLength(0)
    })

    it('does not affect sent items', () => {
      const { result } = renderHook(() => useFeedback())
      act(() => result.current.addItem(makeFeedbackItem()))
      act(() => result.current.markAsSent())
      act(() => result.current.addItem(makeFeedbackItem()))
      act(() => result.current.clearPending())
      expect(result.current.sentItems).toHaveLength(1)
      expect(result.current.pendingItems).toHaveLength(0)
    })
  })
})
