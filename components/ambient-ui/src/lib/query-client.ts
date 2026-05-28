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

export function makeQueryClient() {
  return new QueryClient({
    defaultOptions: queryConfig,
    queryCache: new QueryCache(),
    mutationCache: new MutationCache(),
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
