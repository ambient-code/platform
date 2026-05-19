import type { PaginatedResponse, PaginationParams } from '@/types/api/common'
import type { PaginatedResult } from '../ports/types'
import { DEFAULT_PAGE_SIZE } from '@/types/api/common'

export function toPaginatedResult<T>(
  response: PaginatedResponse<T>,
  fetchPage: (params: PaginationParams) => Promise<PaginatedResponse<T>>,
): PaginatedResult<T> {
  const limit = response.limit ?? DEFAULT_PAGE_SIZE
  const nextOffset = (response.offset ?? 0) + limit

  return {
    items: response.items,
    totalCount: response.totalCount,
    hasMore: response.hasMore,
    nextPage: response.hasMore
      ? async () => {
          const params: PaginationParams = { offset: nextOffset, limit }
          if (response.continue) {
            params.continue = response.continue
          }
          const next = await fetchPage(params)
          return toPaginatedResult(next, fetchPage)
        }
      : undefined,
  }
}
