'use client'

import { useQuery } from '@tanstack/react-query'
import { queryKeys } from './query-keys'

type AgentNameEntry = {
  id: string
  name: string
  displayName: string | null
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
