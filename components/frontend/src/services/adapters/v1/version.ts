import * as versionApi from '../../api/version'
import type { VersionPort } from '../../ports/version'

type VersionApi = typeof versionApi

export function createVersionAdapter(api: VersionApi): VersionPort {
  return {
    getVersion: api.getVersion,
  }
}

export const versionAdapter = createVersionAdapter(versionApi)
