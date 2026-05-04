import * as gerritApi from '../../api/gerrit-auth'
import type { GerritPort } from '../../ports/gerrit'

type GerritApi = typeof gerritApi

export function createGerritAdapter(api: GerritApi): GerritPort {
  return {
    getGerritInstances: api.getGerritInstances,
    getGerritInstanceStatus: api.getGerritInstanceStatus,
    connectGerrit: api.connectGerrit,
    disconnectGerrit: api.disconnectGerrit,
    testGerritConnection: api.testGerritConnection,
  }
}

export const gerritAdapter = createGerritAdapter(gerritApi)
