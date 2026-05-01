import type { ReposStatusResponse } from './types'

export type SessionReposPort = {
  getReposStatus: (projectName: string, sessionName: string) => Promise<ReposStatusResponse>
}
