'use client'

import { useQuery } from '@tanstack/react-query'

export type ConnectionStatus = 'connected' | 'disconnected' | 'checking'

type UseConnectionStatusReturn = {
  status: ConnectionStatus
  lastChecked: string | null
}

async function checkHealth(): Promise<{ ok: boolean }> {
  const response = await fetch('/api/healthz')
  if (!response.ok) {
    throw new Error(`Health check failed: ${response.status}`)
  }
  return { ok: true }
}

export function useConnectionStatus(): UseConnectionStatusReturn {
  const { data, isError, isPending } = useQuery({
    queryKey: ['connection-status'],
    queryFn: checkHealth,
    refetchInterval: 10_000,
    retry: false,
  })

  if (isPending) {
    return { status: 'checking', lastChecked: null }
  }

  if (isError || !data?.ok) {
    return { status: 'disconnected', lastChecked: new Date().toISOString() }
  }

  return { status: 'connected', lastChecked: new Date().toISOString() }
}
