import { renderHook, waitFor, act } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { createProjectsAdapter } from '../../adapters/projects';
import { createProjectAccessAdapter } from '../../adapters/project-access';
import {
  useProjectsPaginated,
  useProjects,
  useProject,
  useCreateProject,
  useUpdateProject,
  useDeleteProject,
  useProjectPermissions,
  useAddProjectPermission,
  useRemoveProjectPermission,
  useProjectIntegrationStatus,
  useProjectMcpServers,
  useUpdateProjectMcpServers,
} from '../use-projects';
import { useProjectAccess } from '../use-project-access';
import { createWrapper } from './test-utils';

const recordedProject = { name: 'my-project', displayName: 'My Project' };

const recordedPaginatedResponse = {
  items: [recordedProject, { name: 'other', displayName: 'Other' }],
  totalCount: 2,
  hasMore: false,
  limit: 20,
  offset: 0,
};

describe('integration: hook → projectsAdapter → fakeApi', () => {
  function createFakeProjectsApi() {
    return {
      listProjectsPaginated: vi.fn().mockResolvedValue(recordedPaginatedResponse),
      getProject: vi.fn().mockResolvedValue(recordedProject),
      createProject: vi.fn().mockResolvedValue({ name: 'new-proj', displayName: 'New' }),
      updateProject: vi.fn().mockResolvedValue({ name: 'my-project', displayName: 'Updated' }),
      deleteProject: vi.fn().mockResolvedValue('deleted'),
      getProjectIntegrationStatus: vi.fn().mockResolvedValue({ github: { connected: true }, gitlab: { connected: false } }),
      getProjectMcpServers: vi.fn().mockResolvedValue({ custom: {} }),
      updateProjectMcpServers: vi.fn().mockResolvedValue({ custom: {} }),
      listProjects: vi.fn(),
      getProjectAccess: vi.fn(),
      getProjectPermissions: vi.fn(),
      addProjectPermission: vi.fn(),
      removeProjectPermission: vi.fn(),
    };
  }

  it('useProjectsPaginated: API paginated response flows through adapter to hook', async () => {
    const fakeApi = createFakeProjectsApi();
    const adapter = createProjectsAdapter(fakeApi);

    const { result } = renderHook(
      () => useProjectsPaginated({ limit: 20 }, adapter),
      { wrapper: createWrapper() },
    );

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.listProjectsPaginated).toHaveBeenCalledWith({ limit: 20 });
    expect(result.current.data?.items).toHaveLength(2);
    expect(result.current.data?.totalCount).toBe(2);
    expect(result.current.data?.hasMore).toBe(false);
    expect(result.current.data?.nextPage).toBeUndefined();
  });

  it('useProjectsPaginated: nextPage is defined when hasMore is true', async () => {
    const fakeApi = createFakeProjectsApi();
    fakeApi.listProjectsPaginated.mockResolvedValueOnce({
      ...recordedPaginatedResponse,
      hasMore: true,
    });
    const adapter = createProjectsAdapter(fakeApi);

    const { result } = renderHook(
      () => useProjectsPaginated({}, adapter),
      { wrapper: createWrapper() },
    );

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data?.hasMore).toBe(true);
    expect(result.current.data?.nextPage).toBeDefined();
  });

  it('useProjects: returns items array from paginated response', async () => {
    const fakeApi = createFakeProjectsApi();
    const adapter = createProjectsAdapter(fakeApi);

    const { result } = renderHook(
      () => useProjects(adapter),
      { wrapper: createWrapper() },
    );

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toHaveLength(2);
    expect(result.current.data?.[0].name).toBe('my-project');
  });

  it('useProject: single project flows through', async () => {
    const fakeApi = createFakeProjectsApi();
    const adapter = createProjectsAdapter(fakeApi);

    const { result } = renderHook(
      () => useProject('my-project', adapter),
      { wrapper: createWrapper() },
    );

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.getProject).toHaveBeenCalledWith('my-project');
    expect(result.current.data?.displayName).toBe('My Project');
  });

  it('useCreateProject: create flows through', async () => {
    const fakeApi = createFakeProjectsApi();
    const adapter = createProjectsAdapter(fakeApi);

    const { result } = renderHook(
      () => useCreateProject(adapter),
      { wrapper: createWrapper() },
    );

    act(() => {
      result.current.mutate({ name: 'new-proj', displayName: 'New' });
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.createProject).toHaveBeenCalledWith({ name: 'new-proj', displayName: 'New' });
    expect(result.current.data?.name).toBe('new-proj');
  });

  it('useUpdateProject: update flows through', async () => {
    const fakeApi = createFakeProjectsApi();
    const adapter = createProjectsAdapter(fakeApi);

    const { result } = renderHook(
      () => useUpdateProject(adapter),
      { wrapper: createWrapper() },
    );

    act(() => {
      result.current.mutate({ name: 'my-project', data: { displayName: 'Updated' } });
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.updateProject).toHaveBeenCalledWith('my-project', { displayName: 'Updated' });
  });

  it('useDeleteProject: delete flows through', async () => {
    const fakeApi = createFakeProjectsApi();
    const adapter = createProjectsAdapter(fakeApi);

    const { result } = renderHook(
      () => useDeleteProject(adapter),
      { wrapper: createWrapper() },
    );

    act(() => {
      result.current.mutate('my-project');
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.deleteProject).toHaveBeenCalledWith('my-project');
  });

  it('useProjectIntegrationStatus: integration status flows through', async () => {
    const fakeApi = createFakeProjectsApi();
    const adapter = createProjectsAdapter(fakeApi);

    const { result } = renderHook(
      () => useProjectIntegrationStatus('my-project', adapter),
      { wrapper: createWrapper() },
    );

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.getProjectIntegrationStatus).toHaveBeenCalledWith('my-project');
    expect(result.current.data).toEqual({ github: { connected: true }, gitlab: { connected: false } });
  });

  it('useProjectMcpServers: MCP servers config flows through', async () => {
    const fakeApi = createFakeProjectsApi();
    const adapter = createProjectsAdapter(fakeApi);

    const { result } = renderHook(
      () => useProjectMcpServers('my-project', adapter),
      { wrapper: createWrapper() },
    );

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.getProjectMcpServers).toHaveBeenCalledWith('my-project');
    expect(result.current.data).toEqual({ custom: {} });
  });

  it('useUpdateProjectMcpServers: MCP servers update flows through', async () => {
    const fakeApi = createFakeProjectsApi();
    const adapter = createProjectsAdapter(fakeApi);

    const { result } = renderHook(
      () => useUpdateProjectMcpServers('my-project', adapter),
      { wrapper: createWrapper() },
    );

    const config = { custom: { 'my-server': { command: 'node', args: ['server.js'] } } };
    act(() => {
      result.current.mutate(config);
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.updateProjectMcpServers).toHaveBeenCalledWith('my-project', config);
  });
});

describe('integration: hook → projectAccessAdapter → fakeApi (method name mapping)', () => {
  function createFakeProjectAccessApi() {
    return {
      getProjectAccess: vi.fn().mockResolvedValue({ project: 'proj', allowed: true, userRole: 'admin' }),
      getProjectPermissions: vi.fn().mockResolvedValue([{ subjectType: 'user', subjectName: 'user1', role: 'admin' }]),
      addProjectPermission: vi.fn().mockResolvedValue({ subjectType: 'user', subjectName: 'user2', role: 'view' }),
      removeProjectPermission: vi.fn().mockResolvedValue(undefined),
    };
  }

  it('useProjectAccess: getProjectAccess → getAccess mapping', async () => {
    const fakeApi = createFakeProjectAccessApi();
    const adapter = createProjectAccessAdapter(fakeApi);

    const { result } = renderHook(
      () => useProjectAccess('proj', adapter),
      { wrapper: createWrapper() },
    );

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.getProjectAccess).toHaveBeenCalledWith('proj');
    expect(result.current.data?.allowed).toBe(true);
    expect(result.current.data?.userRole).toBe('admin');
  });

  it('useProjectPermissions: getProjectPermissions → getPermissions mapping', async () => {
    const fakeApi = createFakeProjectAccessApi();
    const adapter = createProjectAccessAdapter(fakeApi);

    const { result } = renderHook(
      () => useProjectPermissions('proj', adapter),
      { wrapper: createWrapper() },
    );

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.getProjectPermissions).toHaveBeenCalledWith('proj');
    expect(result.current.data).toHaveLength(1);
  });

  it('useAddProjectPermission: addProjectPermission → addPermission mapping', async () => {
    const fakeApi = createFakeProjectAccessApi();
    const adapter = createProjectAccessAdapter(fakeApi);

    const { result } = renderHook(
      () => useAddProjectPermission(adapter),
      { wrapper: createWrapper() },
    );

    act(() => {
      result.current.mutate({
        projectName: 'proj',
        permission: { subjectType: 'user', subjectName: 'user2', role: 'view' },
      });
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.addProjectPermission).toHaveBeenCalledWith('proj', { subjectType: 'user', subjectName: 'user2', role: 'view' });
    expect(result.current.data).toEqual({ subjectType: 'user', subjectName: 'user2', role: 'view' });
  });

  it('useRemoveProjectPermission: removeProjectPermission → removePermission mapping', async () => {
    const fakeApi = createFakeProjectAccessApi();
    const adapter = createProjectAccessAdapter(fakeApi);

    const { result } = renderHook(
      () => useRemoveProjectPermission(adapter),
      { wrapper: createWrapper() },
    );

    act(() => {
      result.current.mutate({ projectName: 'proj', subjectType: 'user', subjectName: 'user1' });
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.removeProjectPermission).toHaveBeenCalledWith('proj', 'user', 'user1');
  });
});
