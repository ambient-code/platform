import type { ListBranchesResponse } from './types'

type RepoParams = {
  repo: string
  ref: string
  path: string
}

type TreeEntry = {
  type: string
  name?: string
  path?: string
  sha?: string
}

type TreeResponse = {
  entries: TreeEntry[]
  sha?: string
}

export type RepoPort = {
  getRepoBlob: (projectName: string, params: RepoParams) => Promise<Response>
  getRepoTree: (projectName: string, params: RepoParams) => Promise<TreeResponse>
  checkFileExists: (projectName: string, params: RepoParams) => Promise<boolean>
  listRepoBranches: (projectName: string, repo: string) => Promise<ListBranchesResponse>
}

export type { RepoParams, TreeEntry, TreeResponse }
