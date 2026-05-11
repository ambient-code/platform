/**
 * Common API types and utilities
 */

export type ApiResponse<T> = {
  data: T;
  error?: never;
};

export type ApiError = {
  error: string;
  code?: string;
  details?: Record<string, unknown>;
};

export type ApiResult<T> = ApiResponse<T> | ApiError;

/**
 * Pagination request parameters
 */
export type PaginationParams = {
  limit?: number;
  offset?: number;
  search?: string;
  continue?: string;
  phase?: string;
  userId?: string;
  sortBy?: string;
  sortDirection?: 'asc' | 'desc';
};

/**
 * Paginated response structure from the backend
 */
export type PaginatedResponse<T> = {
  items: T[];
  totalCount: number;
  limit: number;
  offset: number;
  hasMore: boolean;
  continue?: string;
  nextOffset?: number;
};

/**
 * Default pagination values
 */
export const DEFAULT_PAGE_SIZE = 20;
export const MAX_PAGE_SIZE = 100;

export function isApiError<T>(result: ApiResult<T>): result is ApiError {
  return 'error' in result && result.error !== undefined;
}

export function isApiSuccess<T>(result: ApiResult<T>): result is ApiResponse<T> {
  return 'data' in result && !('error' in result);
}

export class ApiClientError extends Error {
  constructor(
    message: string,
    public code?: string,
    public details?: Record<string, unknown>
  ) {
    super(message);
    this.name = 'ApiClientError';
  }
}
