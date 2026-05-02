import { describe, it, expect, vi } from 'vitest'
import { createGitHubAdapter } from '../github'

describe('githubAdapter', () => {
  it('delegates getGitHubStatus to API', async () => {
    const fakeApi = {
      getGitHubStatus: vi.fn().mockResolvedValue({ installed: true, installationId: 123 }),
      connectGitHub: vi.fn(),
      disconnectGitHub: vi.fn(),
      listGitHubForks: vi.fn(),
      createGitHubFork: vi.fn(),
      getPRDiff: vi.fn(),
      createPullRequest: vi.fn(),
      saveGitHubPAT: vi.fn(),
      getGitHubPATStatus: vi.fn(),
      deleteGitHubPAT: vi.fn(),
    }
    const adapter = createGitHubAdapter(fakeApi)

    const result = await adapter.getGitHubStatus()

    expect(result.installed).toBe(true)
    expect(result.installationId).toBe(123)
  })

  it('delegates listGitHubForks to API', async () => {
    const fakeApi = {
      getGitHubStatus: vi.fn(),
      connectGitHub: vi.fn(),
      disconnectGitHub: vi.fn(),
      listGitHubForks: vi.fn().mockResolvedValue([{ name: 'fork', fullName: 'user/fork', owner: 'user', url: 'https://github.com/user/fork', defaultBranch: 'main', private: false, createdAt: '', updatedAt: '' }]),
      createGitHubFork: vi.fn(),
      getPRDiff: vi.fn(),
      createPullRequest: vi.fn(),
      saveGitHubPAT: vi.fn(),
      getGitHubPATStatus: vi.fn(),
      deleteGitHubPAT: vi.fn(),
    }
    const adapter = createGitHubAdapter(fakeApi)

    const result = await adapter.listGitHubForks('p', 'upstream/repo')

    expect(result).toHaveLength(1)
    expect(result[0].name).toBe('fork')
  })
})
