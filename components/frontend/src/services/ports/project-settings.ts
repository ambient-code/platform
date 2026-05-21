import type { ProjectSettingsResponse, ProjectSettingsPatchRequest } from './types'

export type ProjectSettingsPort = {
  getProjectSettings: (projectName: string) => Promise<ProjectSettingsResponse | null>
  updateProjectSettings: (settingsId: string, patch: ProjectSettingsPatchRequest) => Promise<ProjectSettingsResponse>
}
