import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { secretsAdapter } from '../adapters/secrets';
import type { SecretsPort } from '../ports/secrets';
import type { Secret } from '../ports/types';

export function useSecretsList(projectName: string, port: SecretsPort = secretsAdapter) {
  return useQuery({
    queryKey: ['secrets', 'list', projectName],
    queryFn: () => port.getSecretsList(projectName),
    enabled: !!projectName,
  });
}

export function useSecretsConfig(projectName: string, port: SecretsPort = secretsAdapter) {
  return useQuery({
    queryKey: ['secrets', 'config', projectName],
    queryFn: () => port.getSecretsConfig(projectName),
    enabled: !!projectName,
  });
}

export function useSecretsValues(projectName: string, port: SecretsPort = secretsAdapter) {
  return useQuery({
    queryKey: ['secrets', 'values', projectName],
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
      queryClient.invalidateQueries({ queryKey: ['secrets', 'config', projectName] });
      queryClient.invalidateQueries({ queryKey: ['secrets', 'values', projectName] });
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
      queryClient.invalidateQueries({ queryKey: ['secrets', 'values', projectName] });
    },
  });
}

export function useIntegrationSecrets(projectName: string, port: SecretsPort = secretsAdapter) {
  return useQuery({
    queryKey: ['integration-secrets', projectName],
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
      queryClient.invalidateQueries({ queryKey: ['integration-secrets', projectName] });
    },
  });
}
