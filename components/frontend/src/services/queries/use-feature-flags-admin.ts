import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { featureFlagsAdapter } from '../adapters/feature-flags';
import type { FeatureFlagsPort } from '../ports/feature-flags';
import { BACKEND_VERSION } from './query-keys';

export const featureFlagKeys = {
  all: [BACKEND_VERSION, 'feature-flags'] as const,
  list: (projectName: string) => [...featureFlagKeys.all, 'list', projectName] as const,
  detail: (projectName: string, flagName: string) =>
    [...featureFlagKeys.all, 'detail', projectName, flagName] as const,
  evaluate: (projectName: string, flagName: string) =>
    [...featureFlagKeys.all, 'evaluate', projectName, flagName] as const,
};

export function useFeatureFlags(projectName: string, port: FeatureFlagsPort = featureFlagsAdapter) {
  return useQuery({
    queryKey: featureFlagKeys.list(projectName),
    queryFn: () => port.getFeatureFlags(projectName),
    enabled: !!projectName,
    refetchInterval: (query) => (query.state.status === 'error' ? false : 30000),
    staleTime: 10000,
  });
}

export function useWorkspaceFlag(projectName: string, flagName: string, port: FeatureFlagsPort = featureFlagsAdapter) {
  const { data, isLoading, error } = useQuery({
    queryKey: featureFlagKeys.evaluate(projectName, flagName),
    queryFn: () => port.evaluateFeatureFlag(projectName, flagName),
    enabled: !!projectName && !!flagName,
    staleTime: 15000,
    refetchInterval: (query) => (query.state.status === 'error' ? false : 30000),
  });

  return {
    enabled: data?.enabled ?? false,
    source: data?.source,
    isLoading,
    error,
  };
}

export function useFeatureFlag(projectName: string, flagName: string, port: FeatureFlagsPort = featureFlagsAdapter) {
  return useQuery({
    queryKey: featureFlagKeys.detail(projectName, flagName),
    queryFn: () => port.getFeatureFlag(projectName, flagName),
    enabled: !!projectName && !!flagName,
  });
}

export function useToggleFeatureFlag(port: FeatureFlagsPort = featureFlagsAdapter) {
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
        ? port.enableFeatureFlag(projectName, flagName)
        : port.disableFeatureFlag(projectName, flagName),
    onSuccess: (_, { projectName, flagName }) => {
      queryClient.invalidateQueries({ queryKey: featureFlagKeys.list(projectName) });
      queryClient.invalidateQueries({
        queryKey: featureFlagKeys.evaluate(projectName, flagName),
      });
    },
  });
}

export function useSetFeatureFlagOverride(port: FeatureFlagsPort = featureFlagsAdapter) {
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
    }) => port.setFeatureFlagOverride(projectName, flagName, enabled),
    onSuccess: (_, { projectName, flagName }) => {
      queryClient.invalidateQueries({ queryKey: featureFlagKeys.list(projectName) });
      queryClient.invalidateQueries({
        queryKey: featureFlagKeys.evaluate(projectName, flagName),
      });
    },
  });
}

export function useRemoveFeatureFlagOverride(port: FeatureFlagsPort = featureFlagsAdapter) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      projectName,
      flagName,
    }: {
      projectName: string;
      flagName: string;
    }) => port.removeFeatureFlagOverride(projectName, flagName),
    onSuccess: (_, { projectName, flagName }) => {
      queryClient.invalidateQueries({ queryKey: featureFlagKeys.list(projectName) });
      queryClient.invalidateQueries({
        queryKey: featureFlagKeys.evaluate(projectName, flagName),
      });
    },
  });
}

export function useEnableFeatureFlag(port: FeatureFlagsPort = featureFlagsAdapter) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      projectName,
      flagName,
    }: {
      projectName: string;
      flagName: string;
    }) => port.enableFeatureFlag(projectName, flagName),
    onSuccess: (_, { projectName, flagName }) => {
      queryClient.invalidateQueries({ queryKey: featureFlagKeys.list(projectName) });
      queryClient.invalidateQueries({
        queryKey: featureFlagKeys.evaluate(projectName, flagName),
      });
    },
  });
}

export function useDisableFeatureFlag(port: FeatureFlagsPort = featureFlagsAdapter) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      projectName,
      flagName,
    }: {
      projectName: string;
      flagName: string;
    }) => port.disableFeatureFlag(projectName, flagName),
    onSuccess: (_, { projectName, flagName }) => {
      queryClient.invalidateQueries({ queryKey: featureFlagKeys.list(projectName) });
      queryClient.invalidateQueries({
        queryKey: featureFlagKeys.evaluate(projectName, flagName),
      });
    },
  });
}
