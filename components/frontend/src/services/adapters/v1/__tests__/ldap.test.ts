import { describe, it, expect, vi } from 'vitest'
import { createLdapAdapter } from '../ldap'

describe('ldapAdapter', () => {
  it('delegates searchUsers to API', async () => {
    const fakeApi = {
      searchUsers: vi.fn().mockResolvedValue([{ uid: 'alice', fullName: 'Alice', email: 'alice@test.com', title: 'Eng', githubUsername: 'alice', groups: [] }]),
      searchGroups: vi.fn(),
      getUser: vi.fn(),
    }
    const adapter = createLdapAdapter(fakeApi)

    const result = await adapter.searchUsers('ali')

    expect(result).toHaveLength(1)
    expect(result[0].uid).toBe('alice')
  })

  it('delegates searchGroups to API', async () => {
    const fakeApi = {
      searchUsers: vi.fn(),
      searchGroups: vi.fn().mockResolvedValue([{ name: 'eng', description: 'Engineering' }]),
      getUser: vi.fn(),
    }
    const adapter = createLdapAdapter(fakeApi)

    const result = await adapter.searchGroups('eng')

    expect(result).toHaveLength(1)
    expect(result[0].name).toBe('eng')
  })
})
