import type { DomainProject, ListParams, PaginatedResult } from '@/domain/types'

export type ProjectsPort = {
  list: (params?: ListParams) => Promise<PaginatedResult<DomainProject>>
  get: (projectId: string) => Promise<DomainProject>
}
