import * as secretsApi from '../../api/secrets'
import type { SecretsPort } from '../../ports/secrets'

type SecretsApi = typeof secretsApi

export function createSecretsAdapter(api: SecretsApi): SecretsPort {
  return {
    getSecretsList: api.getSecretsList,
    getSecretsConfig: api.getSecretsConfig,
    getSecretsValues: api.getSecretsValues,
    updateSecretsConfig: api.updateSecretsConfig,
    updateSecrets: api.updateSecrets,
    getIntegrationSecrets: api.getIntegrationSecrets,
    updateIntegrationSecrets: api.updateIntegrationSecrets,
  }
}

export const secretsAdapter = createSecretsAdapter(secretsApi)
