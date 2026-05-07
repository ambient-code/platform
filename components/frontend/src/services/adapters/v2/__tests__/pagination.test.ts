import { describe, it, expect, vi } from 'vitest';
import { toListOptions, fromSdkList } from '../pagination';
import { DEFAULT_PAGE_SIZE } from '@/types/api/common';

describe('toListOptions', () => {
  it('uses defaults when no params provided', () => {
    const opts = toListOptions();
    expect(opts).toEqual({ page: 1, size: DEFAULT_PAGE_SIZE, search: undefined });
  });

  it('uses defaults when empty params provided', () => {
    const opts = toListOptions({});
    expect(opts).toEqual({ page: 1, size: DEFAULT_PAGE_SIZE, search: undefined });
  });

  it('calculates page 1 for offset 0', () => {
    const opts = toListOptions({ offset: 0, limit: 10 });
    expect(opts).toEqual({ page: 1, size: 10, search: undefined });
  });

  it('calculates page 2 for offset equal to limit', () => {
    const opts = toListOptions({ offset: 20, limit: 20 });
    expect(opts).toEqual({ page: 2, size: 20, search: undefined });
  });

  it('calculates page correctly for non-aligned offset', () => {
    const opts = toListOptions({ offset: 45, limit: 20 });
    expect(opts.page).toBe(3);
  });

  it('passes search parameter through', () => {
    const opts = toListOptions({ search: 'test-query' });
    expect(opts.search).toBe('test-query');
  });
});

describe('fromSdkList', () => {
  const identity = <T>(x: T) => x;

  it('transforms items using the transform function', () => {
    const list = { items: [1, 2, 3], total: 3, page: 1, size: 10 };
    const result = fromSdkList(list, (n) => n * 2, vi.fn());
    expect(result.items).toEqual([2, 4, 6]);
  });

  it('sets totalCount from list total', () => {
    const list = { items: ['a'], total: 50, page: 1, size: 10 };
    const result = fromSdkList(list, identity, vi.fn());
    expect(result.totalCount).toBe(50);
  });

  it('sets hasMore false when all items fit in one page', () => {
    const list = { items: [1, 2], total: 2, page: 1, size: 10 };
    const result = fromSdkList(list, identity, vi.fn());
    expect(result.hasMore).toBe(false);
    expect(result.nextPage).toBeUndefined();
  });

  it('sets hasMore true when more pages exist', () => {
    const list = { items: [1, 2, 3], total: 30, page: 1, size: 10 };
    const result = fromSdkList(list, identity, vi.fn());
    expect(result.hasMore).toBe(true);
    expect(result.nextPage).toBeDefined();
  });

  it('calls fetchPage with correct offset and limit for nextPage', async () => {
    const fetchPage = vi.fn().mockResolvedValue({ items: [], totalCount: 0, hasMore: false });
    const list = { items: [1], total: 30, page: 2, size: 10 };
    const result = fromSdkList(list, identity, fetchPage);

    await result.nextPage!();

    expect(fetchPage).toHaveBeenCalledWith({ offset: 20, limit: 10 });
  });

  it('handles empty list', () => {
    const list = { items: [], total: 0, page: 1, size: 10 };
    const result = fromSdkList(list, identity, vi.fn());
    expect(result.items).toEqual([]);
    expect(result.totalCount).toBe(0);
    expect(result.hasMore).toBe(false);
  });

  it('handles last page exactly', () => {
    const list = { items: [1, 2, 3, 4, 5], total: 15, page: 3, size: 5 };
    const result = fromSdkList(list, identity, vi.fn());
    expect(result.hasMore).toBe(false);
    expect(result.nextPage).toBeUndefined();
  });
});
