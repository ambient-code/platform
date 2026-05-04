import type { GitLabStatus, GitLabConnectRequest } from './types'

export type GitLabPort = {
  getGitLabStatus: () => Promise<GitLabStatus>
  connectGitLab: (data: GitLabConnectRequest) => Promise<void>
  disconnectGitLab: () => Promise<void>
}
