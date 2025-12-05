/**
 * Workspace Container API service
 * Handles workspace container settings for ADR-0006 agent isolation
 */

import { apiClient } from './client';
import type { WorkspaceContainerSettings } from '@/types/project-settings';

/**
 * Get workspace container settings for a project
 */
export async function getWorkspaceContainerSettings(
  projectName: string
): Promise<WorkspaceContainerSettings> {
  return apiClient.get<WorkspaceContainerSettings>(
    `/projects/${projectName}/workspace-container`
  );
}

/**
 * Update workspace container settings for a project
 */
export async function updateWorkspaceContainerSettings(
  projectName: string,
  settings: WorkspaceContainerSettings
): Promise<void> {
  await apiClient.put<void, WorkspaceContainerSettings>(
    `/projects/${projectName}/workspace-container`,
    settings
  );
}
