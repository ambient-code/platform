import { describe, it, expect, vi } from 'vitest'
import { createProjectSettingsAdapter } from '../project-settings'
import type { ProjectSettingsResponse } from '@/services/api/projects'

const fakeSettings: ProjectSettingsResponse = {
  id: 'ps-123',
  project_id: 'test-project',
  runner_image: 'quay.io/custom/runner:v1',
  runner_image_pull_secret: 'my-pull-secret',
}

function makeFakeApi() {
  return {
    listProjectsPaginated: vi.fn(),
    listProjects: vi.fn(),
    getProject: vi.fn(),
    createProject: vi.fn(),
    updateProject: vi.fn(),
    deleteProject: vi.fn(),
    getProjectPermissions: vi.fn(),
    addProjectPermission: vi.fn(),
    removeProjectPermission: vi.fn(),
    getProjectIntegrationStatus: vi.fn(),
    getProjectMcpServers: vi.fn(),
    updateProjectMcpServers: vi.fn(),
    getProjectAccess: vi.fn(),
    getProjectSettings: vi.fn().mockResolvedValue(fakeSettings),
    updateProjectSettings: vi.fn().mockResolvedValue(fakeSettings),
  }
}

describe('projectSettingsAdapter', () => {
  it('delegates getProjectSettings to API', async () => {
    const api = makeFakeApi()
    const adapter = createProjectSettingsAdapter(api)

    const result = await adapter.getProjectSettings('test-project')

    expect(result).toEqual(fakeSettings)
    expect(api.getProjectSettings).toHaveBeenCalledWith('test-project')
  })

  it('returns null when no project settings exist', async () => {
    const api = makeFakeApi()
    api.getProjectSettings.mockResolvedValue(null)
    const adapter = createProjectSettingsAdapter(api)

    const result = await adapter.getProjectSettings('empty-project')

    expect(result).toBeNull()
    expect(api.getProjectSettings).toHaveBeenCalledWith('empty-project')
  })

  it('delegates updateProjectSettings to API', async () => {
    const api = makeFakeApi()
    const adapter = createProjectSettingsAdapter(api)
    const patch = { runner_image: 'quay.io/custom/runner:v2' }

    const result = await adapter.updateProjectSettings('ps-123', patch)

    expect(result).toEqual(fakeSettings)
    expect(api.updateProjectSettings).toHaveBeenCalledWith('ps-123', patch)
  })

  it('propagates errors from getProjectSettings', async () => {
    const api = makeFakeApi()
    api.getProjectSettings.mockRejectedValue(new Error('Not found'))
    const adapter = createProjectSettingsAdapter(api)

    await expect(adapter.getProjectSettings('bad-project')).rejects.toThrow('Not found')
  })

  it('propagates errors from updateProjectSettings', async () => {
    const api = makeFakeApi()
    api.updateProjectSettings.mockRejectedValue(new Error('Validation failed'))
    const adapter = createProjectSettingsAdapter(api)

    await expect(
      adapter.updateProjectSettings('ps-123', { runner_image: '' })
    ).rejects.toThrow('Validation failed')
  })
})
