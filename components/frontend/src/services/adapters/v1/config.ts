import * as configApi from '../../api/config'
import type { ConfigPort } from '../../ports/config'

type ConfigApi = typeof configApi

export function createConfigAdapter(api: ConfigApi): ConfigPort {
  return {
    getLoadingTips: api.getLoadingTips,
  }
}

export const configAdapter = createConfigAdapter(configApi)
