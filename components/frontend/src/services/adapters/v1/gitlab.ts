import * as gitlabApi from '../../api/gitlab-auth'
import type { GitLabPort } from '../../ports/gitlab'

type GitLabApi = typeof gitlabApi

export function createGitLabAdapter(api: GitLabApi): GitLabPort {
  return {
    getGitLabStatus: api.getGitLabStatus,
    connectGitLab: api.connectGitLab,
    disconnectGitLab: api.disconnectGitLab,
  }
}

export const gitlabAdapter = createGitLabAdapter(gitlabApi)
