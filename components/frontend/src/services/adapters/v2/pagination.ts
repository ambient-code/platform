import type { ListOptions } from '@/lib/sdk';
import type { PaginationParams } from '@/types/api/common';
import type { PaginatedResult } from '@/services/ports/types';
import { DEFAULT_PAGE_SIZE } from '@/types/api/common';

export function toListOptions(params?: PaginationParams): ListOptions {
  const limit = params?.limit ?? DEFAULT_PAGE_SIZE;
  const offset = params?.offset ?? 0;
  const page = Math.floor(offset / limit) + 1;
  return {
    page,
    size: limit,
    search: params?.search,
  };
}

export function fromSdkList<S, T>(
  list: { items: S[]; total: number; page: number; size: number },
  transform: (item: S) => T,
  fetchPage: (params: PaginationParams) => Promise<PaginatedResult<T>>,
): PaginatedResult<T> {
  const items = list.items.map(transform);
  const offset = (list.page - 1) * list.size;
  const hasMore = offset + list.size < list.total;
  return {
    items,
    totalCount: list.total,
    hasMore,
    nextPage: hasMore
      ? () => fetchPage({ offset: offset + list.size, limit: list.size })
      : undefined,
  };
}
