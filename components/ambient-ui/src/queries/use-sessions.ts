'use client'

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import type { SessionsPort } from '@/ports/sessions'
import type { DomainSession, ListParams, SessionPhase } from '@/domain/types'
import { createSessionsAdapter } from '@/adapters/sdk-sessions'
import { queryKeys } from './query-keys'

const TRANSITIONING_PHASES: ReadonlySet<SessionPhase> = new Set([
  'Pending',
  'Creating',
  'Stopping',
])

const ACTIVE_PHASES: ReadonlySet<SessionPhase> = new Set([
  'Running',
])

const TERMINAL_PHASES: ReadonlySet<SessionPhase> = new Set([
  'Completed',
  'Failed',
  'Stopped',
])

function getPollingInterval(sessions: DomainSession[] | undefined): number | false {
  if (!sessions || sessions.length === 0) {
    return 15000
  }

  const hasTransitioning = sessions.some(s => TRANSITIONING_PHASES.has(s.phase))
  if (hasTransitioning) {
    return 1000
  }

  const hasActive = sessions.some(s => ACTIVE_PHASES.has(s.phase))
  if (hasActive) {
    return 3000
  }

  const allTerminal = sessions.every(s => TERMINAL_PHASES.has(s.phase))
  if (allTerminal) {
    return false
  }

  return 3000
}

let defaultPort: SessionsPort | null = null

function getDefaultPort(): SessionsPort {
  if (!defaultPort) {
    defaultPort = createSessionsAdapter()
  }
  return defaultPort
}

export function useSessions(
  projectId: string,
  params?: ListParams,
  port?: SessionsPort,
) {
  const adapter = port ?? getDefaultPort()
  return useQuery({
    queryKey: queryKeys.sessions.list(projectId, params),
    queryFn: () => adapter.list(projectId, params),
    enabled: !!projectId,
    refetchInterval: (query) => {
      const result = query.state.data
      return getPollingInterval(result?.items)
    },
  })
}

export function useSession(
  sessionId: string,
  port?: SessionsPort,
) {
  const adapter = port ?? getDefaultPort()
  return useQuery({
    queryKey: queryKeys.sessions.detail(sessionId),
    queryFn: () => adapter.get(sessionId),
    enabled: !!sessionId,
    refetchInterval: (query) => {
      const session = query.state.data
      if (!session) return 3000
      if (TRANSITIONING_PHASES.has(session.phase)) return 1000
      if (TERMINAL_PHASES.has(session.phase)) return false
      return 3000
    },
  })
}

export function useStopSession(port?: SessionsPort) {
  const queryClient = useQueryClient()
  const adapter = port ?? getDefaultPort()

  return useMutation({
    mutationFn: (sessionId: string) => adapter.stop(sessionId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.sessions.all })
    },
  })
}

export function useStartSession(port?: SessionsPort) {
  const queryClient = useQueryClient()
  const adapter = port ?? getDefaultPort()

  return useMutation({
    mutationFn: (sessionId: string) => adapter.start(sessionId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.sessions.all })
    },
  })
}
