/**
 * React Query client configuration
 */

import { QueryClient, DefaultOptions, QueryCache, MutationCache } from '@tanstack/react-query';
import { ApiClientError } from '@/types/api/common';

let sessionExpiredCallback: (() => void) | null = null;

export function onSessionExpired(cb: () => void) {
  sessionExpiredCallback = cb;
}

function handleError(error: unknown) {
  if (error instanceof ApiClientError && error.code === '401') {
    sessionExpiredCallback?.();
  }
}

function shouldRetry(failureCount: number, error: unknown): boolean {
  if (error instanceof ApiClientError && error.code === '401') return false;
  return failureCount < 1;
}

const queryConfig: DefaultOptions = {
  queries: {
    // Stale time: 5 minutes - data is considered fresh for 5 minutes
    staleTime: 5 * 60 * 1000,

    // Cache time: 10 minutes - unused data is garbage collected after 10 minutes
    gcTime: 10 * 60 * 1000,

    retry: shouldRetry,

    // Retry delay with exponential backoff
    retryDelay: (attemptIndex) => Math.min(1000 * 2 ** attemptIndex, 30000),

    // Refetch on window focus in production
    refetchOnWindowFocus: process.env.NODE_ENV === 'production',

    // Don't refetch on mount if data is fresh
    refetchOnMount: false,
  },
  mutations: {
    retry: shouldRetry,
  },
};

/**
 * Creates a new QueryClient instance
 * Use this in server components or for testing
 */
export function makeQueryClient() {
  return new QueryClient({
    defaultOptions: queryConfig,
    queryCache: new QueryCache({ onError: handleError }),
    mutationCache: new MutationCache({ onError: handleError }),
  });
}

/**
 * Browser query client singleton
 * Ensures we only create one client instance in the browser
 */
let browserQueryClient: QueryClient | undefined = undefined;

export function getQueryClient() {
  if (typeof window === 'undefined') {
    // Server: always create a new query client
    return makeQueryClient();
  } else {
    // Browser: reuse the same query client
    if (!browserQueryClient) {
      browserQueryClient = makeQueryClient();
    }
    return browserQueryClient;
  }
}
