import * as githubApi from '../../api/github'
import type { GitHubPort } from '../../ports/github'

type GitHubApi = typeof githubApi

export function createGitHubAdapter(api: GitHubApi): GitHubPort {
  return {
    getGitHubStatus: api.getGitHubStatus,
    connectGitHub: api.connectGitHub,
    disconnectGitHub: api.disconnectGitHub,
    listGitHubForks: api.listGitHubForks,
    createGitHubFork: api.createGitHubFork,
    getPRDiff: api.getPRDiff,
    createPullRequest: api.createPullRequest,
    saveGitHubPAT: api.saveGitHubPAT,
    getGitHubPATStatus: api.getGitHubPATStatus,
    deleteGitHubPAT: api.deleteGitHubPAT,
  }
}

export const githubAdapter = createGitHubAdapter(githubApi)
