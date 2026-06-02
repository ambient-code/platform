import { describe, it, expect, afterEach } from 'vitest'
import { renderHook, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { createElement } from 'react'
import { useCurrentUser } from '../use-current-user'

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  return function Wrapper({ children }: { children: React.ReactNode }) {
    return createElement(QueryClientProvider, { client: queryClient }, children)
  }
}

type FakeMeResponse = {
  authenticated: boolean
  username?: string
  name?: string
  email?: string
  initials?: string
}

function installFakeFetch(response: FakeMeResponse, status = 200) {
  const original = globalThis.fetch
  globalThis.fetch = async (input: RequestInfo | URL) => {
    const url = typeof input === 'string' ? input : input.toString()
    if (url === '/api/me') {
      return {
        ok: status >= 200 && status < 300,
        status,
        json: async () => response,
      } as Response
    }
    return original(input)
  }
  return () => {
    globalThis.fetch = original
  }
}

describe('useCurrentUser', () => {
  let cleanup: () => void

  afterEach(() => {
    cleanup?.()
  })

  it('returns isLoading initially', () => {
    cleanup = installFakeFetch({
      authenticated: true,
      username: 'jsell',
      name: 'John Sell',
      email: 'jsell@redhat.com',
      initials: 'JS',
    })

    const { result } = renderHook(() => useCurrentUser(), {
      wrapper: createWrapper(),
    })

    expect(result.current.isLoading).toBe(true)
    expect(result.current.user).toBeNull()
  })

  it('returns user data when authenticated', async () => {
    cleanup = installFakeFetch({
      authenticated: true,
      username: 'jsell',
      name: 'John Sell',
      email: 'jsell@redhat.com',
      initials: 'JS',
    })

    const { result } = renderHook(() => useCurrentUser(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false)
    })
    expect(result.current.user).toEqual({
      username: 'jsell',
      name: 'John Sell',
      email: 'jsell@redhat.com',
      initials: 'JS',
    })
  })

  it('returns null user when not authenticated', async () => {
    cleanup = installFakeFetch({ authenticated: false })

    const { result } = renderHook(() => useCurrentUser(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false)
    })
    expect(result.current.user).toBeNull()
  })

  it('returns null user when fetch fails', async () => {
    cleanup = installFakeFetch({ authenticated: false }, 500)

    const { result } = renderHook(() => useCurrentUser(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false)
    })
    expect(result.current.user).toBeNull()
  })

  it('uses fallback values for missing optional fields', async () => {
    cleanup = installFakeFetch({
      authenticated: true,
    })

    const { result } = renderHook(() => useCurrentUser(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false)
    })
    expect(result.current.user).toEqual({
      username: 'unknown',
      name: '',
      email: '',
      initials: '?',
    })
  })
})
