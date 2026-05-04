import type { ListModelsResponse } from './types'

export type ModelsPort = {
  getModelsForProject: (projectName: string, provider?: string) => Promise<ListModelsResponse>
}
