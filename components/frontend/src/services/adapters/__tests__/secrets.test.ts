import { describe, it, expect, vi } from 'vitest'
import { createSecretsAdapter } from '../secrets'

describe('secretsAdapter', () => {
  it('delegates getSecretsValues to API', async () => {
    const fakeApi = {
      getSecretsList: vi.fn(),
      getSecretsConfig: vi.fn(),
      getSecretsValues: vi.fn().mockResolvedValue([{ key: 'API_KEY', value: 'abc' }]),
      updateSecretsConfig: vi.fn(),
      updateSecrets: vi.fn(),
      getIntegrationSecrets: vi.fn(),
      updateIntegrationSecrets: vi.fn(),
    }
    const adapter = createSecretsAdapter(fakeApi)

    const result = await adapter.getSecretsValues('p')

    expect(result).toHaveLength(1)
    expect(result[0].key).toBe('API_KEY')
  })

  it('delegates updateSecrets to API', async () => {
    const fakeApi = {
      getSecretsList: vi.fn(),
      getSecretsConfig: vi.fn(),
      getSecretsValues: vi.fn(),
      updateSecretsConfig: vi.fn(),
      updateSecrets: vi.fn().mockResolvedValue(undefined),
      getIntegrationSecrets: vi.fn(),
      updateIntegrationSecrets: vi.fn(),
    }
    const adapter = createSecretsAdapter(fakeApi)
    const secrets = [{ key: 'K', value: 'V' }]

    await adapter.updateSecrets('p', secrets)

    expect(fakeApi.updateSecrets).toHaveBeenCalledWith('p', secrets)
  })
})
