import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import * as secretsApi from '../api/secrets';

export function useSecretsList(projectName: string) {
  return useQuery({
    queryKey: ['secrets', 'list', projectName],
    queryFn: () => secretsApi.getSecretsList(projectName),
    enabled: !!projectName,
  });
}

export function useSecretsConfig(projectName: string) {
  return useQuery({
    queryKey: ['secrets', 'config', projectName],
    queryFn: () => secretsApi.getSecretsConfig(projectName),
    enabled: !!projectName,
  });
}

export function useSecretsValues(projectName: string) {
  return useQuery({
    queryKey: ['secrets', 'values', projectName],
    queryFn: () => secretsApi.getSecretsValues(projectName),
    enabled: !!projectName,
  });
}

export function useUpdateSecretsConfig() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      projectName,
      secretName,
    }: {
      projectName: string;
      secretName: string;
    }) => secretsApi.updateSecretsConfig(projectName, secretName),
    onSuccess: (_, { projectName }) => {
      queryClient.invalidateQueries({ queryKey: ['secrets', 'config', projectName] });
      // Also invalidate values since they come from the configured secret
      queryClient.invalidateQueries({ queryKey: ['secrets', 'values', projectName] });
    },
  });
}

export function useUpdateSecrets() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      projectName,
      secrets,
    }: {
      projectName: string;
      secrets: secretsApi.Secret[];
    }) => secretsApi.updateSecrets(projectName, secrets),
    onSuccess: (_, { projectName }) => {
      queryClient.invalidateQueries({ queryKey: ['secrets', 'values', projectName] });
    },
  });
}

// Integration secrets hooks (ambient-non-vertex-integrations)

export function useIntegrationSecrets(projectName: string) {
  return useQuery({
    queryKey: ['integration-secrets', projectName],
    queryFn: () => secretsApi.getIntegrationSecrets(projectName),
    enabled: !!projectName,
  });
}

export function useUpdateIntegrationSecrets() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      projectName,
      secrets,
    }: {
      projectName: string;
      secrets: secretsApi.Secret[];
    }) => secretsApi.updateIntegrationSecrets(projectName, secrets),
    onSuccess: (_, { projectName }) => {
      queryClient.invalidateQueries({ queryKey: ['integration-secrets', projectName] });
    },
  });
}

// Workspace-level generic secrets hooks (ambient-generic-secrets)

export function useGenericSecrets(projectName: string) {
  return useQuery({
    queryKey: ['generic-secrets', projectName],
    queryFn: () => secretsApi.getGenericSecrets(projectName),
    enabled: !!projectName,
  });
}

export function useUpdateGenericSecrets() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      projectName,
      secrets,
    }: {
      projectName: string;
      secrets: secretsApi.Secret[];
    }) => secretsApi.updateGenericSecrets(projectName, secrets),
    onSuccess: (_, { projectName }) => {
      queryClient.invalidateQueries({ queryKey: ['generic-secrets', projectName] });
    },
  });
}

// User-level generic secrets hooks (user-generic-secrets)

export function useUserGenericSecrets() {
  return useQuery({
    queryKey: ['user-generic-secrets'],
    queryFn: () => secretsApi.getUserGenericSecrets(),
  });
}

export function useUpdateUserGenericSecrets() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ secrets }: { secrets: secretsApi.Secret[] }) =>
      secretsApi.updateUserGenericSecrets(secrets),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['user-generic-secrets'] });
    },
  });
}
