import { QueryClient, type DefaultOptions, QueryCache, MutationCache } from '@tanstack/react-query'

const queryConfig: DefaultOptions = {
  queries: {
    staleTime: 5 * 60 * 1000,
    gcTime: 10 * 60 * 1000,
    retry: 1,
    retryDelay: (attemptIndex) => Math.min(1000 * 2 ** attemptIndex, 30000),
    refetchOnWindowFocus: process.env.NODE_ENV === 'production',
    refetchOnMount: false,
  },
  mutations: {
    retry: 1,
  },
}

function isAuthError(error: unknown): boolean {
  if (error instanceof Error && error.message.includes('401')) return true
  if (typeof error === 'object' && error !== null && 'status' in error) {
    return (error as { status: number }).status === 401
  }
  return false
}

let redirecting = false

function handleAuthError() {
  if (redirecting || typeof window === 'undefined') return
  redirecting = true
  const returnTo = encodeURIComponent(window.location.pathname + window.location.search)
  window.location.href = `/api/auth/sso/login?returnTo=${returnTo}`
}

export function makeQueryClient() {
  return new QueryClient({
    defaultOptions: queryConfig,
    queryCache: new QueryCache({
      onError: (error) => {
        if (isAuthError(error)) handleAuthError()
      },
    }),
    mutationCache: new MutationCache({
      onError: (error) => {
        if (isAuthError(error)) handleAuthError()
      },
    }),
  })
}

let browserQueryClient: QueryClient | undefined = undefined

export function getQueryClient() {
  if (typeof window === 'undefined') {
    return makeQueryClient()
  } else {
    if (!browserQueryClient) {
      browserQueryClient = makeQueryClient()
    }
    return browserQueryClient
  }
}
