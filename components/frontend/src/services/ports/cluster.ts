import type { ClusterInfo } from './types'

export type ClusterPort = {
  getClusterInfo: () => Promise<ClusterInfo>
}
