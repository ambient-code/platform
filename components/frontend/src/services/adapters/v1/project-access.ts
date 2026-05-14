import * as projectsApi from '../../api/projects'
import type { ProjectAccessPort } from '../../ports/project-access'

type ProjectAccessApi = Pick<typeof projectsApi, 'getProjectAccess' | 'getProjectPermissions' | 'addProjectPermission' | 'removeProjectPermission'>

export function createProjectAccessAdapter(api: ProjectAccessApi): ProjectAccessPort {
  return {
    getAccess: api.getProjectAccess,
    getPermissions: api.getProjectPermissions,
    addPermission: api.addProjectPermission,
    removePermission: api.removeProjectPermission,
  }
}

export const projectAccessAdapter = createProjectAccessAdapter(projectsApi)
