import { describe, it, expect, vi } from 'vitest'
import { createRepoAdapter } from '../repo'

describe('repoAdapter', () => {
  it('delegates getRepoTree to API', async () => {
    const fakeApi = {
      getRepoBlob: vi.fn(),
      getRepoTree: vi.fn().mockResolvedValue({ entries: [{ type: 'blob', name: 'README.md', path: 'README.md' }] }),
      checkFileExists: vi.fn(),
      listRepoBranches: vi.fn(),
    }
    const adapter = createRepoAdapter(fakeApi)

    const result = await adapter.getRepoTree('p', { repo: 'test/repo', ref: 'main', path: '/' })

    expect(result.entries).toHaveLength(1)
    expect(result.entries[0].name).toBe('README.md')
  })

  it('delegates checkFileExists to API', async () => {
    const fakeApi = {
      getRepoBlob: vi.fn(),
      getRepoTree: vi.fn(),
      checkFileExists: vi.fn().mockResolvedValue(true),
      listRepoBranches: vi.fn(),
    }
    const adapter = createRepoAdapter(fakeApi)

    const result = await adapter.checkFileExists('p', { repo: 'test/repo', ref: 'main', path: 'README.md' })

    expect(result).toBe(true)
  })

  it('delegates listRepoBranches to API', async () => {
    const fakeApi = {
      getRepoBlob: vi.fn(),
      getRepoTree: vi.fn(),
      checkFileExists: vi.fn(),
      listRepoBranches: vi.fn().mockResolvedValue({ branches: [{ name: 'main' }, { name: 'develop' }] }),
    }
    const adapter = createRepoAdapter(fakeApi)

    const result = await adapter.listRepoBranches('p', 'test/repo')

    expect(result.branches).toHaveLength(2)
  })
})
