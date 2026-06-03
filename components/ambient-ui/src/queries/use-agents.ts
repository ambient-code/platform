'use client'

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import type { AgentsPort } from '@/ports/agents'
import type { DomainAgentCreateRequest, DomainAgentUpdateRequest, ListParams } from '@/domain/types'
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
  projectId: string,
  agentId: string,
  port?: AgentsPort,
) {
  const adapter = port ?? getDefaultPort()
  return useQuery({
    queryKey: queryKeys.agents.detail(agentId),
    queryFn: () => adapter.get(projectId, agentId),
    enabled: !!projectId && !!agentId,
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

export function useCreateAgent(port?: AgentsPort) {
  const queryClient = useQueryClient()
  const adapter = port ?? getDefaultPort()

  return useMutation({
    mutationFn: ({ projectId, request }: { projectId: string; request: DomainAgentCreateRequest }) =>
      adapter.create(projectId, request),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.agents.all })
    },
  })
}

export function useUpdateAgent(port?: AgentsPort) {
  const queryClient = useQueryClient()
  const adapter = port ?? getDefaultPort()

  return useMutation({
    mutationFn: ({ projectId, agentId, request }: { projectId: string; agentId: string; request: DomainAgentUpdateRequest }) =>
      adapter.update(projectId, agentId, request),
    onSuccess: (updatedAgent, { agentId }) => {
      queryClient.setQueryData(queryKeys.agents.detail(agentId), updatedAgent)
      queryClient.invalidateQueries({ queryKey: queryKeys.agents.lists() })
    },
  })
}

export function useDeleteAgent(port?: AgentsPort) {
  const queryClient = useQueryClient()
  const adapter = port ?? getDefaultPort()

  return useMutation({
    mutationFn: ({ projectId, agentId }: { projectId: string; agentId: string }) =>
      adapter.delete(projectId, agentId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.agents.all })
    },
  })
}
