/**
 * Feature Flags Admin API
 * Workspace-scoped feature flag management with Unleash fallback
 */

import { apiClient } from './client';

export type Tag = {
  type: string;
  value: string;
};

export type EnvState = {
  name: string;
  enabled: boolean;
};

export type FeatureToggle = {
  name: string;
  description?: string;
  enabled: boolean;
  type?: string;
  stale?: boolean;
  tags?: Tag[];
  environments?: EnvState[];
  source: 'workspace-override' | 'unleash' | 'default';
  overrideEnabled?: boolean | null; // null if no override, true/false if overridden
};

type FeatureToggleListResponse = {
  features: FeatureToggle[];
};

type ToggleResponse = {
  message: string;
  flag: string;
  enabled: boolean;
  source: string;
};

type EvaluateResponse = {
  flag: string;
  enabled: boolean;
  source: 'workspace-override' | 'unleash' | 'default';
  error?: string;
};

/**
 * Get all feature flags for a project with workspace override status
 */
export async function getFeatureFlags(projectName: string): Promise<FeatureToggle[]> {
  const response = await apiClient.get<FeatureToggleListResponse>(
    `/projects/${projectName}/feature-flags`
  );
  return response.features || [];
}

/**
 * Evaluate a feature flag for a workspace (ConfigMap override > Unleash default)
 */
export async function evaluateFeatureFlag(
  projectName: string,
  flagName: string
): Promise<EvaluateResponse> {
  return apiClient.get<EvaluateResponse>(
    `/projects/${projectName}/feature-flags/evaluate/${flagName}`
  );
}

/**
 * Get details for a specific feature flag from Unleash
 */
export async function getFeatureFlag(
  projectName: string,
  flagName: string
): Promise<FeatureToggle> {
  return apiClient.get<FeatureToggle>(
    `/projects/${projectName}/feature-flags/${flagName}`
  );
}

/**
 * Set a workspace-scoped override for a feature flag
 */
export async function setFeatureFlagOverride(
  projectName: string,
  flagName: string,
  enabled: boolean
): Promise<ToggleResponse> {
  return apiClient.put<ToggleResponse>(
    `/projects/${projectName}/feature-flags/${flagName}/override`,
    { enabled }
  );
}

/**
 * Remove a workspace-scoped override (revert to Unleash default)
 */
export async function removeFeatureFlagOverride(
  projectName: string,
  flagName: string
): Promise<ToggleResponse> {
  return apiClient.delete<ToggleResponse>(
    `/projects/${projectName}/feature-flags/${flagName}/override`
  );
}

/**
 * Enable a feature flag for this workspace (sets override to true)
 */
export async function enableFeatureFlag(
  projectName: string,
  flagName: string
): Promise<ToggleResponse> {
  return apiClient.post<ToggleResponse>(
    `/projects/${projectName}/feature-flags/${flagName}/enable`
  );
}

/**
 * Disable a feature flag for this workspace (sets override to false)
 */
export async function disableFeatureFlag(
  projectName: string,
  flagName: string
): Promise<ToggleResponse> {
  return apiClient.post<ToggleResponse>(
    `/projects/${projectName}/feature-flags/${flagName}/disable`
  );
}
