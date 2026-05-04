import type { RunnerType } from './types'

export type RunnerTypesPort = {
  getRunnerTypes: (projectName: string) => Promise<RunnerType[]>
}
