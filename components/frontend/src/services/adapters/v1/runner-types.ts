import * as runnerTypesApi from '../../api/runner-types'
import type { RunnerTypesPort } from '../../ports/runner-types'

type RunnerTypesApi = Pick<typeof runnerTypesApi, 'getRunnerTypes'>

export function createRunnerTypesAdapter(api: RunnerTypesApi): RunnerTypesPort {
  return {
    getRunnerTypes: api.getRunnerTypes,
  }
}

export const runnerTypesAdapter = createRunnerTypesAdapter(runnerTypesApi)
