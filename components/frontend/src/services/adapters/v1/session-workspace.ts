import * as workspaceApi from '../../api/workspace'
import type { SessionWorkspacePort } from '../../ports/session-workspace'

type WorkspaceApi = typeof workspaceApi

export function createSessionWorkspaceAdapter(api: WorkspaceApi): SessionWorkspacePort {
  return {
    listWorkspace: api.listWorkspace,
    readFile: api.readWorkspaceFile,
    writeFile: api.writeWorkspaceFile,
    getGitHubDiff: api.getSessionGitHubDiff,
    pushToGitHub: api.pushSessionToGitHub,
    abandonChanges: api.abandonSessionChanges,
    getGitMergeStatus: api.getGitMergeStatus,
    gitCreateBranch: api.gitCreateBranch,
    gitListBranches: api.gitListBranches,
    gitStatus: api.gitStatus,
    configureGitRemote: api.configureGitRemote,
  }
}

export const sessionWorkspaceAdapter = createSessionWorkspaceAdapter(workspaceApi)
