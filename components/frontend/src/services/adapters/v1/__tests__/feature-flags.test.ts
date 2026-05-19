import { describe, it, expect, vi } from 'vitest'
import { createFeatureFlagsAdapter } from '../feature-flags'

describe('featureFlagsAdapter', () => {
  it('delegates getFeatureFlags to API', async () => {
    const fakeApi = {
      getFeatureFlags: vi.fn().mockResolvedValue([{ name: 'feature-x', enabled: true, source: 'unleash' }]),
      evaluateFeatureFlag: vi.fn(),
      getFeatureFlag: vi.fn(),
      setFeatureFlagOverride: vi.fn(),
      removeFeatureFlagOverride: vi.fn(),
      enableFeatureFlag: vi.fn(),
      disableFeatureFlag: vi.fn(),
    }
    const adapter = createFeatureFlagsAdapter(fakeApi)

    const result = await adapter.getFeatureFlags('p')

    expect(result).toHaveLength(1)
    expect(result[0].name).toBe('feature-x')
  })

  it('delegates evaluateFeatureFlag to API', async () => {
    const fakeApi = {
      getFeatureFlags: vi.fn(),
      evaluateFeatureFlag: vi.fn().mockResolvedValue({ flag: 'feature-x', enabled: true, source: 'workspace-override' }),
      getFeatureFlag: vi.fn(),
      setFeatureFlagOverride: vi.fn(),
      removeFeatureFlagOverride: vi.fn(),
      enableFeatureFlag: vi.fn(),
      disableFeatureFlag: vi.fn(),
    }
    const adapter = createFeatureFlagsAdapter(fakeApi)

    const result = await adapter.evaluateFeatureFlag('p', 'feature-x')

    expect(result.enabled).toBe(true)
    expect(result.source).toBe('workspace-override')
  })
})
