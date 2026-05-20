import * as projectsApi from '../../api/projects'
import type { ProjectSettingsPort } from '../../ports/project-settings'

type ProjectsApi = typeof projectsApi

export function createProjectSettingsAdapter(api: ProjectsApi): ProjectSettingsPort {
  return {
    getProjectSettings: api.getProjectSettings,
    updateProjectSettings: api.updateProjectSettings,
  }
}

export const projectSettingsAdapter = createProjectSettingsAdapter(projectsApi)
