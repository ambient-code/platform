import { describe, it, expect, vi } from 'vitest'
import { toPaginatedResult } from '../pagination'

describe('toPaginatedResult', () => {
  it('transforms a response with no more pages', () => {
    const response = { items: [{ id: 1 }, { id: 2 }], totalCount: 2, limit: 20, offset: 0, hasMore: false }
    const fetchPage = vi.fn()

    const result = toPaginatedResult(response, fetchPage)

    expect(result.items).toEqual([{ id: 1 }, { id: 2 }])
    expect(result.totalCount).toBe(2)
    expect(result.hasMore).toBe(false)
    expect(result.nextPage).toBeUndefined()
    expect(fetchPage).not.toHaveBeenCalled()
  })

  it('provides nextPage when hasMore is true', async () => {
    const page1 = { items: [{ id: 1 }], totalCount: 3, limit: 1, offset: 0, hasMore: true }
    const page2 = { items: [{ id: 2 }], totalCount: 3, limit: 1, offset: 1, hasMore: true }
    const page3 = { items: [{ id: 3 }], totalCount: 3, limit: 1, offset: 2, hasMore: false }
    const fetchPage = vi.fn()
      .mockResolvedValueOnce(page2)
      .mockResolvedValueOnce(page3)

    const result1 = toPaginatedResult(page1, fetchPage)
    expect(result1.nextPage).toBeDefined()

    const result2 = await result1.nextPage!()
    expect(fetchPage).toHaveBeenCalledWith({ offset: 1, limit: 1 })
    expect(result2.items).toEqual([{ id: 2 }])
    expect(result2.nextPage).toBeDefined()

    const result3 = await result2.nextPage!()
    expect(fetchPage).toHaveBeenCalledWith({ offset: 2, limit: 1 })
    expect(result3.items).toEqual([{ id: 3 }])
    expect(result3.nextPage).toBeUndefined()
  })

  it('uses DEFAULT_PAGE_SIZE when limit is missing', () => {
    const response = { items: [], totalCount: 100, limit: 0, offset: 0, hasMore: true }
    const fetchPage = vi.fn()

    const result = toPaginatedResult(response, fetchPage)
    expect(result.nextPage).toBeDefined()
  })
})
