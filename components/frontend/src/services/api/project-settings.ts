import { apiClient } from "./client";
import type {
  ProjectSettingsCR,
  UpdateProjectSettingsRequest,
} from "@/types/project-settings";

export async function getProjectSettings(
  projectName: string
): Promise<ProjectSettingsCR> {
  return apiClient.get<ProjectSettingsCR>(
    `/projects/${projectName}/project-settings`
  );
}

export async function updateProjectSettings(
  projectName: string,
  data: UpdateProjectSettingsRequest
): Promise<ProjectSettingsCR> {
  return apiClient.put<ProjectSettingsCR, UpdateProjectSettingsRequest>(
    `/projects/${projectName}/project-settings`,
    data
  );
}
