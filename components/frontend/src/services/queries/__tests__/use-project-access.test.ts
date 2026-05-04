import { renderHook, waitFor } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { projectAccessKeys, useProjectAccess } from '../use-project-access';
import type { ProjectAccessPort } from '../../ports/project-access';
import { createWrapper, createTestQueryClient } from './test-utils';
import { BACKEND_VERSION } from '../query-keys';

function createFakeProjectAccessPort(
  overrides?: Partial<ProjectAccessPort>,
): ProjectAccessPort {
  return {
    getAccess: vi.fn().mockResolvedValue({
      project: 'test-project',
      allowed: true,
      userRole: 'admin',
    }),
    getPermissions: vi.fn().mockResolvedValue([
      { subjectType: 'user', subjectName: 'user1', role: 'admin' },
    ]),
    addPermission: vi.fn().mockResolvedValue({
      subjectType: 'user',
      subjectName: 'user2',
      role: 'view',
    }),
    removePermission: vi.fn().mockResolvedValue(undefined),
    ...overrides,
  };
}

describe('projectAccessKeys', () => {
  it('includes BACKEND_VERSION prefix', () => {
    expect(projectAccessKeys.all[0]).toBe(BACKEND_VERSION);
  });

  it('generates correct query keys', () => {
    expect(projectAccessKeys.all).toEqual([BACKEND_VERSION, 'project-access']);
    expect(projectAccessKeys.forProject('my-project')).toEqual([
      BACKEND_VERSION,
      'project-access',
      'my-project',
    ]);
  });
});

describe('useProjectAccess', () => {
  it('calls port.getAccess and returns data', async () => {
    const fakePort = createFakeProjectAccessPort();
    const { result } = renderHook(
      () => useProjectAccess('test-project', fakePort),
      { wrapper: createWrapper() },
    );

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.getAccess).toHaveBeenCalledWith('test-project');
    expect(result.current.data).toEqual({
      project: 'test-project',
      allowed: true,
      userRole: 'admin',
    });
  });

  it('uses the correct query key', async () => {
    const queryClient = createTestQueryClient();
    const fakePort = createFakeProjectAccessPort();
    const { result } = renderHook(
      () => useProjectAccess('test-project', fakePort),
      { wrapper: createWrapper(queryClient) },
    );

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    const cache = queryClient.getQueryCache().findAll();
    expect(cache).toHaveLength(1);
    expect(cache[0].queryKey).toEqual(projectAccessKeys.forProject('test-project'));
  });

  it('is disabled when projectName is empty', () => {
    const fakePort = createFakeProjectAccessPort();
    const { result } = renderHook(
      () => useProjectAccess('', fakePort),
      { wrapper: createWrapper() },
    );

    expect(result.current.fetchStatus).toBe('idle');
    expect(fakePort.getAccess).not.toHaveBeenCalled();
  });

  it('calls port with correct project name', async () => {
    const fakePort = createFakeProjectAccessPort();
    const { result } = renderHook(
      () => useProjectAccess('my-project', fakePort),
      { wrapper: createWrapper() },
    );

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.getAccess).toHaveBeenCalledWith('my-project');
  });
});
