import type { ProjectAPI } from 'ambient-sdk'
import type { ProjectsPort, ProjectCreateInput } from '@/ports/projects'
import type { DomainProject, ListParams, PaginatedResult } from '@/domain/types'
import { mapSdkProjectToDomain } from './mappers'
import { getProjectAPI } from './sdk-client'

function buildSdkListOptions(params?: ListParams) {
  return {
    page: params?.page ?? 1,
    size: params?.size ?? 20,
    search: params?.search,
    orderBy: params?.orderBy,
  }
}

function createSdkProjectsAdapter(api: ProjectAPI): ProjectsPort {
  return {
    async list(params?: ListParams): Promise<PaginatedResult<DomainProject>> {
      const opts = buildSdkListOptions(params)
      const result = await api.list(opts)
      const items = result.items.map(mapSdkProjectToDomain)
      const page = opts.page
      const size = opts.size
      return {
        items,
        total: result.total,
        page,
        size,
        hasMore: page * size < result.total,
      }
    },

    async get(projectId: string): Promise<DomainProject> {
      const project = await api.get(projectId)
      return mapSdkProjectToDomain(project)
    },

    async create(input: ProjectCreateInput): Promise<DomainProject> {
      const project = await api.create({ name: input.name, description: input.description })
      return mapSdkProjectToDomain(project)
    },
  }
}

export function createProjectsAdapter(api?: ProjectAPI): ProjectsPort {
  return createSdkProjectsAdapter(api ?? getProjectAPI())
}
