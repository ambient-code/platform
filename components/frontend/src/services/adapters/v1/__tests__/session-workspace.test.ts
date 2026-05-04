import { describe, it, expect, vi } from 'vitest'
import { createSessionWorkspaceAdapter } from '../session-workspace'

function makeFakeApi() {
  return {
    listWorkspace: vi.fn().mockResolvedValue([{ name: 'file.ts', path: '/file.ts', isDir: false, size: 100, modifiedAt: '2024-01-01' }]),
    readWorkspaceFile: vi.fn().mockResolvedValue('file content'),
    writeWorkspaceFile: vi.fn().mockResolvedValue(undefined),
    getSessionGitHubDiff: vi.fn().mockResolvedValue({ files: { added: 5, removed: 2 }, total_added: 5, total_removed: 2 }),
    pushSessionToGitHub: vi.fn().mockResolvedValue(undefined),
    abandonSessionChanges: vi.fn().mockResolvedValue(undefined),
    getGitMergeStatus: vi.fn().mockResolvedValue({ canMergeClean: true, localChanges: 0, remoteCommitsAhead: 0, conflictingFiles: [], remoteBranchExists: true }),
    gitCreateBranch: vi.fn().mockResolvedValue(undefined),
    gitListBranches: vi.fn().mockResolvedValue(['main', 'develop']),
    gitStatus: vi.fn().mockResolvedValue({ branch: 'main', hasChanges: false }),
    configureGitRemote: vi.fn().mockResolvedValue(undefined),
    deleteWorkspaceFile: vi.fn(),
  }
}

describe('sessionWorkspaceAdapter', () => {
  it('delegates listWorkspace to API', async () => {
    const api = makeFakeApi()
    const adapter = createSessionWorkspaceAdapter(api)

    const result = await adapter.listWorkspace('p', 's')

    expect(result).toHaveLength(1)
    expect(result[0].name).toBe('file.ts')
  })

  it('delegates readFile to API', async () => {
    const api = makeFakeApi()
    const adapter = createSessionWorkspaceAdapter(api)

    const result = await adapter.readFile('p', 's', '/file.ts')

    expect(result).toBe('file content')
    expect(api.readWorkspaceFile).toHaveBeenCalledWith('p', 's', '/file.ts')
  })

  it('delegates writeFile to API', async () => {
    const api = makeFakeApi()
    const adapter = createSessionWorkspaceAdapter(api)

    await adapter.writeFile('p', 's', '/file.ts', 'new content')

    expect(api.writeWorkspaceFile).toHaveBeenCalledWith('p', 's', '/file.ts', 'new content')
  })

  it('delegates getGitHubDiff to API', async () => {
    const api = makeFakeApi()
    const adapter = createSessionWorkspaceAdapter(api)

    const result = await adapter.getGitHubDiff('p', 's', 0, '/repo')

    expect(result.total_added).toBe(5)
    expect(result.total_removed).toBe(2)
  })

  it('delegates gitStatus to API', async () => {
    const api = makeFakeApi()
    const adapter = createSessionWorkspaceAdapter(api)

    const result = await adapter.gitStatus('p', 's', '/repo')

    expect(result.branch).toBe('main')
    expect(result.hasChanges).toBe(false)
  })

  it('delegates gitListBranches to API', async () => {
    const api = makeFakeApi()
    const adapter = createSessionWorkspaceAdapter(api)

    const result = await adapter.gitListBranches('p', 's')

    expect(result).toEqual(['main', 'develop'])
  })
})
