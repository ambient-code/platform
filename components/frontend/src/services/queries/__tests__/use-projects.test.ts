import { renderHook, waitFor, act } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import {
  useProjects,
  useProjectsPaginated,
  useProject,
  useCreateProject,
  useUpdateProject,
  useDeleteProject,
  useProjectPermissions,
  useAddProjectPermission,
  useRemoveProjectPermission,
  useProjectIntegrationStatus,
  projectKeys,
} from '../use-projects';
import type { ProjectsPort } from '../../ports/projects';
import type { ProjectAccessPort } from '../../ports/project-access';
import { createWrapper, createTestQueryClient } from './test-utils';
import { BACKEND_VERSION } from '../query-keys';

const mockProject = { name: 'test-project', displayName: 'Test Project' };
const mockPaginatedProjects = {
  items: [
    { name: 'test-project', displayName: 'Test Project' },
    { name: 'other', displayName: 'Other' },
  ],
  totalCount: 2,
  hasMore: false,
  nextPage: undefined,
};

function createFakeProjectsPort(overrides?: Partial<ProjectsPort>): ProjectsPort {
  return {
    listProjects: vi.fn().mockResolvedValue(mockPaginatedProjects),
    getProject: vi.fn().mockResolvedValue(mockProject),
    createProject: vi.fn().mockResolvedValue({ name: 'new-project', displayName: 'New' }),
    updateProject: vi.fn().mockResolvedValue({ name: 'test-project', displayName: 'Updated' }),
    deleteProject: vi.fn().mockResolvedValue('deleted'),
    getProjectIntegrationStatus: vi.fn().mockResolvedValue({ github: { connected: true } }),
    getProjectMcpServers: vi.fn().mockResolvedValue({}),
    updateProjectMcpServers: vi.fn().mockResolvedValue({}),
    ...overrides,
  };
}

function createFakeProjectAccessPort(overrides?: Partial<ProjectAccessPort>): ProjectAccessPort {
  return {
    getAccess: vi.fn().mockResolvedValue({ project: 'test-project', allowed: true, userRole: 'admin' }),
    getPermissions: vi.fn().mockResolvedValue([{ subjectType: 'user', subjectName: 'user1', role: 'admin' }]),
    addPermission: vi.fn().mockResolvedValue({ subjectType: 'user', subjectName: 'user2', role: 'view' }),
    removePermission: vi.fn().mockResolvedValue(undefined),
    ...overrides,
  };
}

describe('projectKeys', () => {
  it('includes BACKEND_VERSION prefix', () => {
    expect(projectKeys.all[0]).toBe(BACKEND_VERSION);
  });

  it('generates correct query keys', () => {
    expect(projectKeys.all).toEqual(['v1', 'projects']);
    expect(projectKeys.lists()).toEqual(['v1', 'projects', 'list']);
    expect(projectKeys.detail('proj')).toEqual(['v1', 'projects', 'detail', 'proj']);
    expect(projectKeys.permissions('proj')).toEqual(['v1', 'projects', 'detail', 'proj', 'permissions']);
    expect(projectKeys.integrationStatus('proj')).toEqual(['v1', 'projects', 'detail', 'proj', 'integration-status']);
  });
});

describe('useProjects', () => {
  it('fetches projects list', async () => {
    const fakePort = createFakeProjectsPort();
    const { result } = renderHook(() => useProjects(fakePort), {
      wrapper: createWrapper(),
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.listProjects).toHaveBeenCalled();
    expect(result.current.data).toHaveLength(2);
  });
});

describe('useProjectsPaginated', () => {
  it('fetches paginated projects', async () => {
    const fakePort = createFakeProjectsPort();
    const { result } = renderHook(
      () => useProjectsPaginated({ limit: 10 }, fakePort),
      { wrapper: createWrapper() },
    );
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.listProjects).toHaveBeenCalledWith({ limit: 10 });
    expect(result.current.data?.items).toHaveLength(2);
    expect(result.current.data?.totalCount).toBe(2);
  });
});

describe('useProject', () => {
  it('fetches a single project', async () => {
    const fakePort = createFakeProjectsPort();
    const { result } = renderHook(() => useProject('test-project', fakePort), {
      wrapper: createWrapper(),
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.getProject).toHaveBeenCalledWith('test-project');
    expect(result.current.data?.name).toBe('test-project');
  });

  it('is disabled when name is empty', () => {
    const fakePort = createFakeProjectsPort();
    const { result } = renderHook(() => useProject('', fakePort), {
      wrapper: createWrapper(),
    });
    expect(result.current.fetchStatus).toBe('idle');
    expect(fakePort.getProject).not.toHaveBeenCalled();
  });

  it('uses the correct query key', async () => {
    const queryClient = createTestQueryClient();
    const fakePort = createFakeProjectsPort();
    const { result } = renderHook(() => useProject('test-project', fakePort), {
      wrapper: createWrapper(queryClient),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    const cache = queryClient.getQueryCache().findAll();
    expect(cache[0].queryKey).toEqual(projectKeys.detail('test-project'));
  });
});

describe('useCreateProject', () => {
  it('creates a project and invalidates list cache', async () => {
    const queryClient = createTestQueryClient();
    const fakePort = createFakeProjectsPort();

    queryClient.setQueryData(projectKeys.lists(), []);

    const { result } = renderHook(() => useCreateProject(fakePort), {
      wrapper: createWrapper(queryClient),
    });

    act(() => {
      result.current.mutate({ name: 'new-project', displayName: 'New' });
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.createProject).toHaveBeenCalledWith({ name: 'new-project', displayName: 'New' });
    expect(result.current.data?.name).toBe('new-project');
  });
});

describe('useUpdateProject', () => {
  it('updates a project', async () => {
    const fakePort = createFakeProjectsPort();
    const { result } = renderHook(() => useUpdateProject(fakePort), {
      wrapper: createWrapper(),
    });

    act(() => {
      result.current.mutate({
        name: 'test-project',
        data: { displayName: 'Updated' },
      });
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.updateProject).toHaveBeenCalledWith('test-project', { displayName: 'Updated' });
    expect(result.current.data?.name).toBe('test-project');
  });
});

describe('useDeleteProject', () => {
  it('deletes a project', async () => {
    const fakePort = createFakeProjectsPort();
    const { result } = renderHook(() => useDeleteProject(fakePort), {
      wrapper: createWrapper(),
    });

    act(() => {
      result.current.mutate('test-project');
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.deleteProject).toHaveBeenCalledWith('test-project');
  });
});

describe('useProjectPermissions', () => {
  it('fetches project permissions', async () => {
    const fakePort = createFakeProjectAccessPort();
    const { result } = renderHook(
      () => useProjectPermissions('test-project', fakePort),
      { wrapper: createWrapper() },
    );
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.getPermissions).toHaveBeenCalledWith('test-project');
    expect(result.current.data).toHaveLength(1);
  });

  it('is disabled when projectName is empty', () => {
    const fakePort = createFakeProjectAccessPort();
    const { result } = renderHook(
      () => useProjectPermissions('', fakePort),
      { wrapper: createWrapper() },
    );
    expect(result.current.fetchStatus).toBe('idle');
    expect(fakePort.getPermissions).not.toHaveBeenCalled();
  });
});

describe('useAddProjectPermission', () => {
  it('adds a permission and invalidates cache', async () => {
    const queryClient = createTestQueryClient();
    const fakePort = createFakeProjectAccessPort();

    queryClient.setQueryData(projectKeys.permissions('test-project'), []);

    const { result } = renderHook(() => useAddProjectPermission(fakePort), {
      wrapper: createWrapper(queryClient),
    });

    act(() => {
      result.current.mutate({
        projectName: 'test-project',
        permission: { subjectType: 'user', subjectName: 'user1', role: 'view' },
      });
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.addPermission).toHaveBeenCalled();
    expect(queryClient.getQueryState(projectKeys.permissions('test-project'))?.isInvalidated).toBe(true);
  });
});

describe('useRemoveProjectPermission', () => {
  it('removes a permission and invalidates cache', async () => {
    const queryClient = createTestQueryClient();
    const fakePort = createFakeProjectAccessPort();

    queryClient.setQueryData(projectKeys.permissions('test-project'), [{ subjectType: 'user', subjectName: 'user1', role: 'admin' }]);

    const { result } = renderHook(() => useRemoveProjectPermission(fakePort), {
      wrapper: createWrapper(queryClient),
    });

    act(() => {
      result.current.mutate({
        projectName: 'test-project',
        subjectType: 'user',
        subjectName: 'user1',
      });
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.removePermission).toHaveBeenCalledWith('test-project', 'user', 'user1');
    expect(queryClient.getQueryState(projectKeys.permissions('test-project'))?.isInvalidated).toBe(true);
  });
});

describe('useProjectIntegrationStatus', () => {
  it('fetches integration status', async () => {
    const fakePort = createFakeProjectsPort();
    const { result } = renderHook(
      () => useProjectIntegrationStatus('test-project', fakePort),
      { wrapper: createWrapper() },
    );
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.getProjectIntegrationStatus).toHaveBeenCalledWith('test-project');
    expect(result.current.data).toEqual({ github: { connected: true } });
  });

  it('is disabled when projectName is empty', () => {
    const fakePort = createFakeProjectsPort();
    const { result } = renderHook(
      () => useProjectIntegrationStatus('', fakePort),
      { wrapper: createWrapper() },
    );
    expect(result.current.fetchStatus).toBe('idle');
    expect(fakePort.getProjectIntegrationStatus).not.toHaveBeenCalled();
  });
});
