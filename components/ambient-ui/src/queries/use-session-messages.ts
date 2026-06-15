'use client'

import { useQuery } from '@tanstack/react-query'
import type { SessionMessagesPort } from '@/ports/session-messages'
import { createSessionMessagesAdapterWithFetch } from '@/adapters/session-messages'
import { queryKeys } from './query-keys'

let defaultPort: SessionMessagesPort | null = null

function getDefaultPort(): SessionMessagesPort {
  if (!defaultPort) {
    defaultPort = createSessionMessagesAdapterWithFetch()
  }
  return defaultPort
}

export function useSessionMessages(
  sessionId: string,
  port?: SessionMessagesPort,
) {
  const adapter = port ?? getDefaultPort()
  return useQuery({
    queryKey: queryKeys.messages.list(sessionId),
    queryFn: () => adapter.list(sessionId, { size: 1000 }),
    enabled: !!sessionId,
    refetchInterval: 3000,
  })
}
