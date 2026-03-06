/* eslint-disable react/display-name, @typescript-eslint/no-unused-vars */
import { renderHook, waitFor, act } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { describe, it, expect, vi } from 'vitest';
import React from 'react';
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

const mockProject = { name: 'test-project', displayName: 'Test Project' };

vi.mock('@/services/api/projects', () => ({
  listProjects: vi.fn().mockResolvedValue([
    { name: 'test-project', displayName: 'Test Project' },
    { name: 'other', displayName: 'Other' },
  ]),
  listProjectsPaginated: vi.fn().mockResolvedValue({
    items: [{ name: 'test-project', displayName: 'Test Project' }],
    totalCount: 1,
  }),
  getProject: vi.fn().mockResolvedValue({ name: 'test-project', displayName: 'Test Project' }),
  createProject: vi.fn().mockResolvedValue({ name: 'new-project', displayName: 'New' }),
  updateProject: vi.fn().mockResolvedValue({ name: 'test-project', displayName: 'Updated' }),
  deleteProject: vi.fn().mockResolvedValue('deleted'),
  getProjectPermissions: vi.fn().mockResolvedValue([{ subject: 'user1', role: 'admin' }]),
  addProjectPermission: vi.fn().mockResolvedValue(undefined),
  removeProjectPermission: vi.fn().mockResolvedValue(undefined),
  getProjectIntegrationStatus: vi.fn().mockResolvedValue({ github: { connected: true } }),
}));

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return ({ children }: { children: React.ReactNode }) =>
    React.createElement(QueryClientProvider, { client: queryClient }, children);
}

describe('projectKeys', () => {
  it('generates correct query keys', () => {
    expect(projectKeys.all).toEqual(['projects']);
    expect(projectKeys.lists()).toEqual(['projects', 'list']);
    expect(projectKeys.detail('proj')).toEqual(['projects', 'detail', 'proj']);
    expect(projectKeys.permissions('proj')).toEqual(['projects', 'detail', 'proj', 'permissions']);
    expect(projectKeys.integrationStatus('proj')).toEqual(['projects', 'detail', 'proj', 'integration-status']);
  });
});

describe('useProjects', () => {
  it('fetches projects list', async () => {
    const wrapper = createWrapper();
    const { result } = renderHook(() => useProjects(), { wrapper });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toHaveLength(2);
  });
});

describe('useProjectsPaginated', () => {
  it('fetches paginated projects', async () => {
    const wrapper = createWrapper();
    const { result } = renderHook(() => useProjectsPaginated({ limit: 10 }), { wrapper });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toEqual({ items: [{ name: 'test-project', displayName: 'Test Project' }], totalCount: 1 });
  });
});

describe('useProject', () => {
  it('fetches a single project', async () => {
    const wrapper = createWrapper();
    const { result } = renderHook(() => useProject('test-project'), { wrapper });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data?.name).toBe('test-project');
  });

  it('is disabled when name is empty', () => {
    const wrapper = createWrapper();
    const { result } = renderHook(() => useProject(''), { wrapper });
    expect(result.current.fetchStatus).toBe('idle');
  });
});

describe('useCreateProject', () => {
  it('creates a project', async () => {
    const wrapper = createWrapper();
    const { result } = renderHook(() => useCreateProject(), { wrapper });

    act(() => {
      result.current.mutate({ name: 'new-project', displayName: 'New' });
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data?.name).toBe('new-project');
  });
});

describe('useUpdateProject', () => {
  it('updates a project', async () => {
    const wrapper = createWrapper();
    const { result } = renderHook(() => useUpdateProject(), { wrapper });

    act(() => {
      result.current.mutate({
        name: 'test-project',
        data: { displayName: 'Updated' },
      });
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data?.name).toBe('test-project');
  });
});

describe('useDeleteProject', () => {
  it('deletes a project', async () => {
    const wrapper = createWrapper();
    const { result } = renderHook(() => useDeleteProject(), { wrapper });

    act(() => {
      result.current.mutate('test-project');
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
  });
});

describe('useProjectPermissions', () => {
  it('fetches project permissions', async () => {
    const wrapper = createWrapper();
    const { result } = renderHook(() => useProjectPermissions('test-project'), { wrapper });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toHaveLength(1);
  });

  it('is disabled when projectName is empty', () => {
    const wrapper = createWrapper();
    const { result } = renderHook(() => useProjectPermissions(''), { wrapper });
    expect(result.current.fetchStatus).toBe('idle');
  });
});

describe('useAddProjectPermission', () => {
  it('adds a permission', async () => {
    const wrapper = createWrapper();
    const { result } = renderHook(() => useAddProjectPermission(), { wrapper });

    act(() => {
      result.current.mutate({
        projectName: 'test-project',
        permission: { subjectType: 'user', subjectName: 'user1', role: 'view' },
      });
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
  });
});

describe('useRemoveProjectPermission', () => {
  it('removes a permission', async () => {
    const wrapper = createWrapper();
    const { result } = renderHook(() => useRemoveProjectPermission(), { wrapper });

    act(() => {
      result.current.mutate({
        projectName: 'test-project',
        subjectType: 'user',
        subjectName: 'user1',
      });
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
  });
});

describe('useProjectIntegrationStatus', () => {
  it('fetches integration status', async () => {
    const wrapper = createWrapper();
    const { result } = renderHook(() => useProjectIntegrationStatus('test-project'), { wrapper });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toEqual({ github: { connected: true } });
  });

  it('is disabled when projectName is empty', () => {
    const wrapper = createWrapper();
    const { result } = renderHook(() => useProjectIntegrationStatus(''), { wrapper });
    expect(result.current.fetchStatus).toBe('idle');
  });
});
