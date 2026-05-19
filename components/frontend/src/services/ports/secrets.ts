import type { Secret, SecretList, SecretsConfig } from './types'

export type SecretsPort = {
  getSecretsList: (projectName: string) => Promise<SecretList>
  getSecretsConfig: (projectName: string) => Promise<SecretsConfig>
  getSecretsValues: (projectName: string) => Promise<Secret[]>
  updateSecretsConfig: (projectName: string, secretName: string) => Promise<void>
  updateSecrets: (projectName: string, secrets: Secret[]) => Promise<void>
  getIntegrationSecrets: (projectName: string) => Promise<Secret[]>
  updateIntegrationSecrets: (projectName: string, secrets: Secret[]) => Promise<void>
}
