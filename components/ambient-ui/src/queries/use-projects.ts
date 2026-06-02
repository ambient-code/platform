'use client'

import { useQuery } from '@tanstack/react-query'
import type { ProjectsPort } from '@/ports/projects'
import type { ListParams } from '@/domain/types'
import { createProjectsAdapter } from '@/adapters/sdk-projects'
import { queryKeys } from './query-keys'

let defaultPort: ProjectsPort | null = null

function getDefaultPort(): ProjectsPort {
  if (!defaultPort) {
    defaultPort = createProjectsAdapter()
  }
  return defaultPort
}

export function useProjects(
  params?: ListParams,
  port?: ProjectsPort,
) {
  const adapter = port ?? getDefaultPort()
  return useQuery({
    queryKey: queryKeys.projects.list(params),
    queryFn: () => adapter.list(params),
    staleTime: 30_000,
    refetchInterval: 30_000,
  })
}

export function useProject(
  projectId: string,
  port?: ProjectsPort,
) {
  const adapter = port ?? getDefaultPort()
  return useQuery({
    queryKey: queryKeys.projects.detail(projectId),
    queryFn: () => adapter.get(projectId),
    enabled: !!projectId,
    staleTime: 30_000,
    refetchInterval: 30_000,
  })
}
