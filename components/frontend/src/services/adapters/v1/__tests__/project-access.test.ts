import { describe, it, expect, vi } from 'vitest'
import { createProjectAccessAdapter } from '../project-access'

describe('projectAccessAdapter', () => {
  it('delegates getAccess to API', async () => {
    const fakeApi = {
      getProjectAccess: vi.fn().mockResolvedValue({ project: 'p', allowed: true, userRole: 'admin' }),
      getProjectPermissions: vi.fn(),
      addProjectPermission: vi.fn(),
      removeProjectPermission: vi.fn(),
    }
    const adapter = createProjectAccessAdapter(fakeApi)

    const result = await adapter.getAccess('p')

    expect(result.allowed).toBe(true)
    expect(result.userRole).toBe('admin')
  })

  it('delegates getPermissions to API', async () => {
    const fakeApi = {
      getProjectAccess: vi.fn(),
      getProjectPermissions: vi.fn().mockResolvedValue([{ subjectType: 'user', subjectName: 'alice', role: 'edit' }]),
      addProjectPermission: vi.fn(),
      removeProjectPermission: vi.fn(),
    }
    const adapter = createProjectAccessAdapter(fakeApi)

    const result = await adapter.getPermissions('p')

    expect(result).toHaveLength(1)
    expect(result[0].subjectName).toBe('alice')
  })
})
