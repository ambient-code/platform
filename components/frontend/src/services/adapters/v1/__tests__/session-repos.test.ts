import { describe, it, expect, vi } from 'vitest'
import { createSessionReposAdapter } from '../session-repos'

describe('sessionReposAdapter', () => {
  it('delegates getReposStatus to API', async () => {
    const fakeApi = {
      getReposStatus: vi.fn().mockResolvedValue({
        repos: [{ url: 'https://github.com/test/repo', name: 'repo', branches: ['main'], currentActiveBranch: 'main', defaultBranch: 'main' }],
      }),
    }
    const adapter = createSessionReposAdapter(fakeApi)

    const result = await adapter.getReposStatus('p', 's')

    expect(result.repos).toHaveLength(1)
    expect(result.repos[0].name).toBe('repo')
    expect(fakeApi.getReposStatus).toHaveBeenCalledWith('p', 's')
  })
})
