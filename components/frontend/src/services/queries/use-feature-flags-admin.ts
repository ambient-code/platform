/**
 * React Query hooks for Feature Flags Admin
 * Workspace-scoped feature flag management with Unleash fallback
 */

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import * as featureFlagsApi from '../api/feature-flags-admin';

export const featureFlagKeys = {
  all: ['feature-flags'] as const,
  list: (projectName: string) => [...featureFlagKeys.all, 'list', projectName] as const,
  detail: (projectName: string, flagName: string) =>
    [...featureFlagKeys.all, 'detail', projectName, flagName] as const,
  evaluate: (projectName: string, flagName: string) =>
    [...featureFlagKeys.all, 'evaluate', projectName, flagName] as const,
};

/**
 * Hook to fetch all feature flags for a project with workspace override status
 */
export function useFeatureFlags(projectName: string) {
  return useQuery({
    queryKey: featureFlagKeys.list(projectName),
    queryFn: () => featureFlagsApi.getFeatureFlags(projectName),
    enabled: !!projectName,
    refetchInterval: 30000, // Refresh every 30s to stay in sync
    staleTime: 10000, // Consider data stale after 10s
  });
}

/**
 * Hook to evaluate a workspace-scoped feature flag
 * Returns the effective value (ConfigMap override > Unleash default)
 */
export function useWorkspaceFlag(projectName: string, flagName: string) {
  const { data, isLoading, error } = useQuery({
    queryKey: featureFlagKeys.evaluate(projectName, flagName),
    queryFn: () => featureFlagsApi.evaluateFeatureFlag(projectName, flagName),
    enabled: !!projectName && !!flagName,
    staleTime: 15000, // 15s cache
    refetchInterval: 30000, // Refresh every 30s
  });

  return {
    enabled: data?.enabled ?? false,
    source: data?.source,
    isLoading,
    error,
  };
}

/**
 * Hook to fetch a specific feature flag from Unleash
 */
export function useFeatureFlag(projectName: string, flagName: string) {
  return useQuery({
    queryKey: featureFlagKeys.detail(projectName, flagName),
    queryFn: () => featureFlagsApi.getFeatureFlag(projectName, flagName),
    enabled: !!projectName && !!flagName,
  });
}

/**
 * Hook to toggle a feature flag (enable or disable) for this workspace
 */
export function useToggleFeatureFlag() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      projectName,
      flagName,
      enable,
    }: {
      projectName: string;
      flagName: string;
      enable: boolean;
    }) =>
      enable
        ? featureFlagsApi.enableFeatureFlag(projectName, flagName)
        : featureFlagsApi.disableFeatureFlag(projectName, flagName),
    onSuccess: (_, { projectName, flagName }) => {
      // Invalidate both list and evaluate queries
      queryClient.invalidateQueries({ queryKey: featureFlagKeys.list(projectName) });
      queryClient.invalidateQueries({
        queryKey: featureFlagKeys.evaluate(projectName, flagName),
      });
    },
  });
}

/**
 * Hook to set a workspace-scoped override for a feature flag
 */
export function useSetFeatureFlagOverride() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      projectName,
      flagName,
      enabled,
    }: {
      projectName: string;
      flagName: string;
      enabled: boolean;
    }) => featureFlagsApi.setFeatureFlagOverride(projectName, flagName, enabled),
    onSuccess: (_, { projectName, flagName }) => {
      queryClient.invalidateQueries({ queryKey: featureFlagKeys.list(projectName) });
      queryClient.invalidateQueries({
        queryKey: featureFlagKeys.evaluate(projectName, flagName),
      });
    },
  });
}

/**
 * Hook to remove a workspace-scoped override (revert to Unleash default)
 */
export function useRemoveFeatureFlagOverride() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      projectName,
      flagName,
    }: {
      projectName: string;
      flagName: string;
    }) => featureFlagsApi.removeFeatureFlagOverride(projectName, flagName),
    onSuccess: (_, { projectName, flagName }) => {
      queryClient.invalidateQueries({ queryKey: featureFlagKeys.list(projectName) });
      queryClient.invalidateQueries({
        queryKey: featureFlagKeys.evaluate(projectName, flagName),
      });
    },
  });
}

/**
 * Hook to enable a feature flag for this workspace
 */
export function useEnableFeatureFlag() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      projectName,
      flagName,
    }: {
      projectName: string;
      flagName: string;
    }) => featureFlagsApi.enableFeatureFlag(projectName, flagName),
    onSuccess: (_, { projectName, flagName }) => {
      queryClient.invalidateQueries({ queryKey: featureFlagKeys.list(projectName) });
      queryClient.invalidateQueries({
        queryKey: featureFlagKeys.evaluate(projectName, flagName),
      });
    },
  });
}

/**
 * Hook to disable a feature flag for this workspace
 */
export function useDisableFeatureFlag() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      projectName,
      flagName,
    }: {
      projectName: string;
      flagName: string;
    }) => featureFlagsApi.disableFeatureFlag(projectName, flagName),
    onSuccess: (_, { projectName, flagName }) => {
      queryClient.invalidateQueries({ queryKey: featureFlagKeys.list(projectName) });
      queryClient.invalidateQueries({
        queryKey: featureFlagKeys.evaluate(projectName, flagName),
      });
    },
  });
}
