import type { FeatureToggle } from './types'

type ToggleResponse = {
  message: string
  flag: string
  enabled: boolean
  source: string
}

type EvaluateResponse = {
  flag: string
  enabled: boolean
  source: 'workspace-override' | 'unleash' | 'default'
  error?: string
}

export type FeatureFlagsPort = {
  getFeatureFlags: (projectName: string) => Promise<FeatureToggle[]>
  evaluateFeatureFlag: (projectName: string, flagName: string) => Promise<EvaluateResponse>
  getFeatureFlag: (projectName: string, flagName: string) => Promise<FeatureToggle>
  setFeatureFlagOverride: (projectName: string, flagName: string, enabled: boolean) => Promise<ToggleResponse>
  removeFeatureFlagOverride: (projectName: string, flagName: string) => Promise<ToggleResponse>
  enableFeatureFlag: (projectName: string, flagName: string) => Promise<ToggleResponse>
  disableFeatureFlag: (projectName: string, flagName: string) => Promise<ToggleResponse>
}
