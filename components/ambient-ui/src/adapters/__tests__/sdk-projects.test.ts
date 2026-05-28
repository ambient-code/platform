import { describe, it, expect } from 'vitest'
import type { Project, ProjectList, ListOptions } from 'ambient-sdk'
import type { ProjectsPort } from '@/ports/projects'
import { createProjectsAdapter } from '../sdk-projects'

function makeSdkProject(overrides: Partial<Project> = {}): Project {
  return {
    id: 'proj-001',
    kind: 'Project',
    href: '/api/ambient/v1/projects/proj-001',
    created_at: '2026-01-10T08:00:00Z',
    updated_at: '2026-01-10T09:00:00Z',
    annotations: '',
    description: 'Test project description',
    labels: '',
    name: 'test-project',
    prompt: '',
    status: 'active',
    ...overrides,
  }
}

// Fake ProjectAPI satisfying the shape the adapter needs
function createFakeProjectAPI(options: {
  projects?: Project[]
  total?: number
  getResult?: Project
}) {
  const projects = options.projects ?? [makeSdkProject()]
  const total = options.total ?? projects.length

  return {
    list: async (): Promise<ProjectList> => ({
      kind: 'ProjectList',
      page: 1,
      size: 20,
      total,
      items: projects,
    }),
    get: async (): Promise<Project> => {
      return options.getResult ?? projects[0]
    },
    // Unused methods — included to satisfy the type shape
    create: async () => makeSdkProject(),
    update: async () => makeSdkProject(),
    delete: async () => undefined,
    listAll: async function* () { yield makeSdkProject() },
  }
}

describe('sdk-projects adapter', () => {
  it('list() returns paginated domain projects', async () => {
    const projects = [
      makeSdkProject({ id: 'proj-001', name: 'first-project' }),
      makeSdkProject({ id: 'proj-002', name: 'second-project' }),
    ]
    const fakeAPI = createFakeProjectAPI({ projects, total: 50 })
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const adapter: ProjectsPort = createProjectsAdapter(fakeAPI as any)

    const result = await adapter.list()

    expect(result.items).toHaveLength(2)
    expect(result.items[0].id).toBe('proj-001')
    expect(result.items[0].name).toBe('first-project')
    expect(result.items[1].id).toBe('proj-002')
    expect(result.total).toBe(50)
    expect(result.page).toBe(1)
    expect(result.size).toBe(20)
    expect(result.hasMore).toBe(true)
  })

  it('list() returns hasMore=false when all items fit on one page', async () => {
    const projects = [makeSdkProject()]
    const fakeAPI = createFakeProjectAPI({ projects, total: 1 })
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const adapter: ProjectsPort = createProjectsAdapter(fakeAPI as any)

    const result = await adapter.list()

    expect(result.hasMore).toBe(false)
  })

  it('list() maps SDK projects to domain projects', async () => {
    const projects = [makeSdkProject({
      description: 'Production environment',
      status: 'active',
    })]
    const fakeAPI = createFakeProjectAPI({ projects })
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const adapter: ProjectsPort = createProjectsAdapter(fakeAPI as any)

    const result = await adapter.list()
    const project = result.items[0]

    expect(project.description).toBe('Production environment')
    expect(project.status).toBe('active')
  })

  it('get() returns a mapped domain project', async () => {
    const getResult = makeSdkProject({ id: 'proj-xyz', name: 'specific-project' })
    const fakeAPI = createFakeProjectAPI({ getResult })
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const adapter: ProjectsPort = createProjectsAdapter(fakeAPI as any)

    const project = await adapter.get('proj-xyz')

    expect(project.id).toBe('proj-xyz')
    expect(project.name).toBe('specific-project')
  })

  it('list() passes custom pagination params', async () => {
    let capturedOpts: ListOptions | undefined
    const fakeAPI = {
      ...createFakeProjectAPI({}),
      list: async (listOpts?: ListOptions): Promise<ProjectList> => {
        capturedOpts = listOpts
        return { kind: 'ProjectList', page: 3, size: 5, total: 50, items: [] }
      },
    }
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const adapter: ProjectsPort = createProjectsAdapter(fakeAPI as any)

    await adapter.list({ page: 3, size: 5 })

    expect(capturedOpts?.page).toBe(3)
    expect(capturedOpts?.size).toBe(5)
  })

  it('list() handles empty description and status as null', async () => {
    const projects = [makeSdkProject({ description: '', status: '' })]
    const fakeAPI = createFakeProjectAPI({ projects })
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const adapter: ProjectsPort = createProjectsAdapter(fakeAPI as any)

    const result = await adapter.list()
    const project = result.items[0]

    expect(project.description).toBeNull()
    expect(project.status).toBeNull()
  })
})
