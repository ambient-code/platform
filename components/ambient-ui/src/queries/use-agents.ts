'use client'

import { useQuery } from '@tanstack/react-query'
import type { AgentsPort } from '@/ports/agents'
import type { ListParams } from '@/domain/types'
import { createAgentsAdapter } from '@/adapters/sdk-agents'
import { queryKeys } from './query-keys'

type AgentNameEntry = {
  id: string
  name: string
  displayName: string | null
}

let defaultPort: AgentsPort | null = null

function getDefaultPort(): AgentsPort {
  if (!defaultPort) {
    defaultPort = createAgentsAdapter()
  }
  return defaultPort
}

export function useAgents(
  projectId: string,
  params?: ListParams,
  port?: AgentsPort,
) {
  const adapter = port ?? getDefaultPort()
  return useQuery({
    queryKey: queryKeys.agents.list(projectId, params),
    queryFn: () => adapter.list(projectId, params),
    enabled: !!projectId,
  })
}

export function useAgent(
  agentId: string,
  port?: AgentsPort,
) {
  const adapter = port ?? getDefaultPort()
  return useQuery({
    queryKey: queryKeys.agents.detail(agentId),
    queryFn: () => adapter.get(agentId),
    enabled: !!agentId,
  })
}

export function useAgentNames(projectId: string) {
  return useQuery({
    queryKey: queryKeys.agents.names(projectId),
    queryFn: async (): Promise<Map<string, string>> => {
      const res = await fetch(`/api/ambient/v1/projects/${encodeURIComponent(projectId)}/agents?size=100`)
      if (!res.ok) return new Map()
      const data: { items?: AgentNameEntry[] } = await res.json()
      const map = new Map<string, string>()
      for (const agent of data.items ?? []) {
        map.set(agent.id, agent.displayName || agent.name)
      }
      return map
    },
    enabled: !!projectId,
    staleTime: 60_000,
  })
}
