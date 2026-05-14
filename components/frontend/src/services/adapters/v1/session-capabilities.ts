import * as sessionsApi from '../../api/sessions'
import type { SessionCapabilitiesPort } from '../../ports/session-capabilities'

type SessionCapabilitiesApi = Pick<typeof sessionsApi, 'getCapabilities'>

export function createSessionCapabilitiesAdapter(api: SessionCapabilitiesApi): SessionCapabilitiesPort {
  return {
    getCapabilities: api.getCapabilities,
  }
}

export const sessionCapabilitiesAdapter = createSessionCapabilitiesAdapter(sessionsApi)
