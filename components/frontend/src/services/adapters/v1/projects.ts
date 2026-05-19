import * as projectsApi from '../../api/projects'
import type { ProjectsPort } from '../../ports/projects'
import { toPaginatedResult } from '../pagination'

type ProjectsApi = typeof projectsApi

export function createProjectsAdapter(api: ProjectsApi): ProjectsPort {
  return {
    listProjects: async (params = {}) => {
      const response = await api.listProjectsPaginated(params)
      return toPaginatedResult(response, (p) => api.listProjectsPaginated(p))
    },
    getProject: api.getProject,
    createProject: api.createProject,
    updateProject: api.updateProject,
    deleteProject: api.deleteProject,
    getProjectIntegrationStatus: api.getProjectIntegrationStatus,
    getProjectMcpServers: api.getProjectMcpServers,
    updateProjectMcpServers: api.updateProjectMcpServers,
  }
}

export const projectsAdapter = createProjectsAdapter(projectsApi)
