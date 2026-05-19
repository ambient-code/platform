import * as flagsApi from '../../api/feature-flags-admin'
import type { FeatureFlagsPort } from '../../ports/feature-flags'

type FeatureFlagsApi = typeof flagsApi

export function createFeatureFlagsAdapter(api: FeatureFlagsApi): FeatureFlagsPort {
  return {
    getFeatureFlags: api.getFeatureFlags,
    evaluateFeatureFlag: api.evaluateFeatureFlag,
    getFeatureFlag: api.getFeatureFlag,
    setFeatureFlagOverride: api.setFeatureFlagOverride,
    removeFeatureFlagOverride: api.removeFeatureFlagOverride,
    enableFeatureFlag: api.enableFeatureFlag,
    disableFeatureFlag: api.disableFeatureFlag,
  }
}

export const featureFlagsAdapter = createFeatureFlagsAdapter(flagsApi)
