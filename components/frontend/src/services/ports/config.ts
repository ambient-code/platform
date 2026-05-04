import type { LoadingTipsResponse } from './types'

export type ConfigPort = {
  getLoadingTips: () => Promise<LoadingTipsResponse>
}
