import type { GerritInstancesResponse, GerritInstanceStatus, GerritConnectRequest, GerritTestRequest, GerritTestResponse } from './types'

export type GerritPort = {
  getGerritInstances: () => Promise<GerritInstancesResponse>
  getGerritInstanceStatus: (instanceName: string) => Promise<GerritInstanceStatus>
  connectGerrit: (data: GerritConnectRequest) => Promise<void>
  disconnectGerrit: (instanceName: string) => Promise<void>
  testGerritConnection: (data: GerritTestRequest) => Promise<GerritTestResponse>
}
