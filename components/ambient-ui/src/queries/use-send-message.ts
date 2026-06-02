'use client'

import { useRef } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import type { SessionMessagesPort } from '@/ports/session-messages'
import { createSessionMessagesAdapterWithFetch } from '@/adapters/session-messages'
import { queryKeys } from './query-keys'

export function useSendMessage(sessionId: string, port?: SessionMessagesPort) {
  const defaultPortRef = useRef<SessionMessagesPort | null>(null)
  if (!defaultPortRef.current && !port) {
    defaultPortRef.current = createSessionMessagesAdapterWithFetch()
  }
  const adapter = port ?? defaultPortRef.current!
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (payload: string) => {
      return adapter.send(sessionId, {
        eventType: 'user',
        payload,
      })
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: queryKeys.messages.list(sessionId),
      })
    },
  })
}
