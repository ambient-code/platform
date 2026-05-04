import * as modelsApi from '../../api/models'
import type { ModelsPort } from '../../ports/models'

type ModelsApi = typeof modelsApi

export function createModelsAdapter(api: ModelsApi): ModelsPort {
  return {
    getModelsForProject: api.getModelsForProject,
  }
}

export const modelsAdapter = createModelsAdapter(modelsApi)
