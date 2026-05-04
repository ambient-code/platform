import * as integrationsApi from '../../api/integrations'
import type { IntegrationsPort } from '../../ports/integrations'

type IntegrationsApi = typeof integrationsApi

export function createIntegrationsAdapter(api: IntegrationsApi): IntegrationsPort {
  return {
    getIntegrationsStatus: api.getIntegrationsStatus,
  }
}

export const integrationsAdapter = createIntegrationsAdapter(integrationsApi)
