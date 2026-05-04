import type { CapabilitiesResponse } from './types'

export type SessionCapabilitiesPort = {
  getCapabilities: (projectName: string, sessionName: string) => Promise<CapabilitiesResponse>
}
