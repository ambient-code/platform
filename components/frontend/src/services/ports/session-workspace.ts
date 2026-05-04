import type { WorkspaceItem, GitMergeStatus, GitStatus } from './types'

export type SessionWorkspacePort = {
  listWorkspace: (projectName: string, sessionName: string, path?: string) => Promise<WorkspaceItem[]>
  readFile: (projectName: string, sessionName: string, path: string) => Promise<string>
  writeFile: (projectName: string, sessionName: string, path: string, content: string) => Promise<void>
  getGitHubDiff: (projectName: string, sessionName: string, repoIndex: number, repoPath: string) => Promise<{ files: { added: number; removed: number }; total_added: number; total_removed: number }>
  pushToGitHub: (projectName: string, sessionName: string, repoIndex: number, repoPath: string) => Promise<void>
  abandonChanges: (projectName: string, sessionName: string, repoIndex: number, repoPath: string) => Promise<void>
  getGitMergeStatus: (projectName: string, sessionName: string, path?: string, branch?: string) => Promise<GitMergeStatus>
  gitCreateBranch: (projectName: string, sessionName: string, branchName: string, path?: string) => Promise<void>
  gitListBranches: (projectName: string, sessionName: string, path?: string) => Promise<string[]>
  gitStatus: (projectName: string, sessionName: string, path: string) => Promise<GitStatus>
  configureGitRemote: (projectName: string, sessionName: string, path: string, remoteUrl: string, branch?: string) => Promise<void>
}
