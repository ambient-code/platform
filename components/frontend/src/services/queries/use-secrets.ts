import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { secretsAdapter } from '../adapters/secrets';
import type { SecretsPort } from '../ports/secrets';
import type { Secret } from '../ports/types';
import { BACKEND_VERSION } from './query-keys';

export const secretsKeys = {
  all: [BACKEND_VERSION, 'secrets'] as const,
  lists: () => [...secretsKeys.all, 'list'] as const,
  list: (projectName: string) => [...secretsKeys.lists(), projectName] as const,
  configs: () => [...secretsKeys.all, 'config'] as const,
  config: (projectName: string) => [...secretsKeys.configs(), projectName] as const,
  values: () => [...secretsKeys.all, 'values'] as const,
  valuesForProject: (projectName: string) => [...secretsKeys.values(), projectName] as const,
  integrations: () => [BACKEND_VERSION, 'integration-secrets'] as const,
  integration: (projectName: string) => [...secretsKeys.integrations(), projectName] as const,
};

export function useSecretsList(projectName: string, port: SecretsPort = secretsAdapter) {
  return useQuery({
    queryKey: secretsKeys.list(projectName),
    queryFn: () => port.getSecretsList(projectName),
    enabled: !!projectName,
  });
}

export function useSecretsConfig(projectName: string, port: SecretsPort = secretsAdapter) {
  return useQuery({
    queryKey: secretsKeys.config(projectName),
    queryFn: () => port.getSecretsConfig(projectName),
    enabled: !!projectName,
  });
}

export function useSecretsValues(projectName: string, port: SecretsPort = secretsAdapter) {
  return useQuery({
    queryKey: secretsKeys.valuesForProject(projectName),
    queryFn: () => port.getSecretsValues(projectName),
    enabled: !!projectName,
  });
}

export function useUpdateSecretsConfig(port: SecretsPort = secretsAdapter) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      projectName,
      secretName,
    }: {
      projectName: string;
      secretName: string;
    }) => port.updateSecretsConfig(projectName, secretName),
    onSuccess: (_, { projectName }) => {
      queryClient.invalidateQueries({ queryKey: secretsKeys.config(projectName) });
      queryClient.invalidateQueries({ queryKey: secretsKeys.valuesForProject(projectName) });
    },
  });
}

export function useUpdateSecrets(port: SecretsPort = secretsAdapter) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      projectName,
      secrets,
    }: {
      projectName: string;
      secrets: Secret[];
    }) => port.updateSecrets(projectName, secrets),
    onSuccess: (_, { projectName }) => {
      queryClient.invalidateQueries({ queryKey: secretsKeys.valuesForProject(projectName) });
    },
  });
}

export function useIntegrationSecrets(projectName: string, port: SecretsPort = secretsAdapter) {
  return useQuery({
    queryKey: secretsKeys.integration(projectName),
    queryFn: () => port.getIntegrationSecrets(projectName),
    enabled: !!projectName,
  });
}

export function useUpdateIntegrationSecrets(port: SecretsPort = secretsAdapter) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      projectName,
      secrets,
    }: {
      projectName: string;
      secrets: Secret[];
    }) => port.updateIntegrationSecrets(projectName, secrets),
    onSuccess: (_, { projectName }) => {
      queryClient.invalidateQueries({ queryKey: secretsKeys.integration(projectName) });
    },
  });
}
