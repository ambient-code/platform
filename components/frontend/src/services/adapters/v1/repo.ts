import * as repoApi from '../../api/repo'
import type { RepoPort } from '../../ports/repo'

type RepoApi = typeof repoApi

export function createRepoAdapter(api: RepoApi): RepoPort {
  return {
    getRepoBlob: api.getRepoBlob,
    getRepoTree: api.getRepoTree,
    checkFileExists: api.checkFileExists,
    listRepoBranches: api.listRepoBranches,
  }
}

export const repoAdapter = createRepoAdapter(repoApi)
