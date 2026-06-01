import { describe, it, expect, beforeEach, afterEach } from 'vitest'
import { renderHook, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { createElement } from 'react'
import { useConnectionStatus } from '../use-connection-status'

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  return function Wrapper({ children }: { children: React.ReactNode }) {
    return createElement(QueryClientProvider, { client: queryClient }, children)
  }
}

type FakeFetchResult = {
  ok: boolean
  status: number
  json: () => Promise<Record<string, string>>
}

function installFakeFetch(result: FakeFetchResult) {
  const original = globalThis.fetch
  globalThis.fetch = async (input: RequestInfo | URL) => {
    const url = typeof input === 'string' ? input : input.toString()
    if (url === '/api/healthz') {
      return result as Response
    }
    return original(input)
  }
  return () => {
    globalThis.fetch = original
  }
}

describe('useConnectionStatus', () => {
  let cleanup: () => void

  afterEach(() => {
    cleanup?.()
  })

  it('returns "checking" as the initial status', () => {
    cleanup = installFakeFetch({
      ok: true,
      status: 200,
      json: async () => ({ status: 'ok' }),
    })

    const { result } = renderHook(() => useConnectionStatus(), {
      wrapper: createWrapper(),
    })

    expect(result.current.status).toBe('checking')
  })

  it('returns "connected" when healthz returns 200', async () => {
    cleanup = installFakeFetch({
      ok: true,
      status: 200,
      json: async () => ({ status: 'ok' }),
    })

    const { result } = renderHook(() => useConnectionStatus(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => {
      expect(result.current.status).toBe('connected')
    })
    expect(result.current.lastChecked).not.toBeNull()
  })

  it('returns "disconnected" when healthz returns an error', async () => {
    cleanup = installFakeFetch({
      ok: false,
      status: 500,
      json: async () => ({ status: 'error' }),
    })

    const { result } = renderHook(() => useConnectionStatus(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => {
      expect(result.current.status).toBe('disconnected')
    })
  })

  it('returns "disconnected" when fetch throws a network error', async () => {
    const original = globalThis.fetch
    globalThis.fetch = async (input: RequestInfo | URL) => {
      const url = typeof input === 'string' ? input : input.toString()
      if (url === '/api/healthz') {
        throw new Error('Network error')
      }
      return original(input)
    }
    cleanup = () => {
      globalThis.fetch = original
    }

    const { result } = renderHook(() => useConnectionStatus(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => {
      expect(result.current.status).toBe('disconnected')
    })
  })
})
