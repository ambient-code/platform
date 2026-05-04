import { describe, it, expect, vi } from 'vitest'
import { createProjectsAdapter } from '../projects'
import type { Project, ListProjectsPaginatedResponse } from '@/types/api'

const fakeProject: Project = {
  name: 'test-project',
  displayName: 'Test Project',
  labels: {},
  annotations: {},
  creationTimestamp: '2024-01-01T00:00:00Z',
  status: 'active',
  isOpenShift: false,
}

const fakePaginatedResponse: ListProjectsPaginatedResponse = {
  items: [fakeProject],
  totalCount: 1,
  limit: 20,
  offset: 0,
  hasMore: false,
}

function makeFakeApi() {
  return {
    listProjectsPaginated: vi.fn().mockResolvedValue(fakePaginatedResponse),
    listProjects: vi.fn().mockResolvedValue([fakeProject]),
    getProject: vi.fn().mockResolvedValue(fakeProject),
    createProject: vi.fn().mockResolvedValue(fakeProject),
    updateProject: vi.fn().mockResolvedValue(fakeProject),
    deleteProject: vi.fn().mockResolvedValue('deleted'),
    getProjectPermissions: vi.fn().mockResolvedValue([]),
    addProjectPermission: vi.fn(),
    removeProjectPermission: vi.fn(),
    getProjectIntegrationStatus: vi.fn().mockResolvedValue({ github: true }),
    getProjectMcpServers: vi.fn().mockResolvedValue({}),
    updateProjectMcpServers: vi.fn().mockResolvedValue({}),
    getProjectAccess: vi.fn(),
  }
}

describe('projectsAdapter', () => {
  it('transforms paginated response to PaginatedResult', async () => {
    const api = makeFakeApi()
    const adapter = createProjectsAdapter(api)

    const result = await adapter.listProjects()

    expect(result.items).toHaveLength(1)
    expect(result.items[0].name).toBe('test-project')
    expect(result.totalCount).toBe(1)
    expect(result.hasMore).toBe(false)
    expect(result.nextPage).toBeUndefined()
  })

  it('provides nextPage when hasMore is true', async () => {
    const api = makeFakeApi()
    api.listProjectsPaginated
      .mockResolvedValueOnce({ ...fakePaginatedResponse, hasMore: true })
      .mockResolvedValueOnce(fakePaginatedResponse)
    const adapter = createProjectsAdapter(api)

    const result = await adapter.listProjects()
    expect(result.nextPage).toBeDefined()
  })

  it('delegates getProject to API', async () => {
    const api = makeFakeApi()
    const adapter = createProjectsAdapter(api)

    const result = await adapter.getProject('test-project')

    expect(result.name).toBe('test-project')
    expect(api.getProject).toHaveBeenCalledWith('test-project')
  })

  it('delegates createProject to API', async () => {
    const api = makeFakeApi()
    const adapter = createProjectsAdapter(api)

    await adapter.createProject({ name: 'new-project' })

    expect(api.createProject).toHaveBeenCalledWith({ name: 'new-project' })
  })

  it('delegates deleteProject to API', async () => {
    const api = makeFakeApi()
    const adapter = createProjectsAdapter(api)

    const result = await adapter.deleteProject('test-project')

    expect(result).toBe('deleted')
  })
})
