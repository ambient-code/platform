import * as clusterApi from '../../api/cluster'
import type { ClusterPort } from '../../ports/cluster'

type ClusterApi = typeof clusterApi

export function createClusterAdapter(api: ClusterApi): ClusterPort {
  return {
    getClusterInfo: api.getClusterInfo,
  }
}

export const clusterAdapter = createClusterAdapter(clusterApi)
