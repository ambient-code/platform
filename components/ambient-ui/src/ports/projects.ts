import type { DomainProject, ListParams, PaginatedResult } from '@/domain/types'

export type ProjectCreateInput = {
  name: string
  description?: string
}

export type ProjectsPort = {
  list: (params?: ListParams) => Promise<PaginatedResult<DomainProject>>
  get: (projectId: string) => Promise<DomainProject>
  create: (input: ProjectCreateInput) => Promise<DomainProject>
}
