import type { IntegrationsStatus } from './types'

export type IntegrationsPort = {
  getIntegrationsStatus: () => Promise<IntegrationsStatus>
}
