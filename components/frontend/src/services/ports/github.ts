import type { GitHubStatus, GitHubFork, CreateForkRequest, PRDiff, CreatePRRequest, GitHubConnectRequest } from './types'

export type GitHubPort = {
  getGitHubStatus: () => Promise<GitHubStatus>
  connectGitHub: (data: GitHubConnectRequest) => Promise<string>
  disconnectGitHub: () => Promise<string>
  listGitHubForks: (projectName?: string, upstreamRepo?: string) => Promise<GitHubFork[]>
  createGitHubFork: (data: CreateForkRequest, projectName?: string) => Promise<GitHubFork>
  getPRDiff: (owner: string, repo: string, prNumber: number, projectName?: string) => Promise<PRDiff>
  createPullRequest: (data: CreatePRRequest, projectName?: string) => Promise<{ url: string; number: number }>
  saveGitHubPAT: (token: string) => Promise<void>
  getGitHubPATStatus: () => Promise<{ configured: boolean; updatedAt?: string }>
  deleteGitHubPAT: () => Promise<void>
}
