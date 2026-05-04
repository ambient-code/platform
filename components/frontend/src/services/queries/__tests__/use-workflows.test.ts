import { renderHook, waitFor } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import {
  workflowKeys,
  useOOTBWorkflows,
  useWorkflowMetadata,
} from '../use-workflows';
import type { WorkflowsPort } from '../../ports/workflows';
import { createWrapper, createTestQueryClient } from './test-utils';
import { BACKEND_VERSION } from '../query-keys';

function createFakeWorkflowsPort(
  overrides?: Partial<WorkflowsPort>,
): WorkflowsPort {
  return {
    listOOTBWorkflows: vi.fn().mockResolvedValue([
      {
        name: 'code-review',
        displayName: 'Code Review',
        description: 'Automated code review workflow',
      },
    ]),
    getWorkflowMetadata: vi.fn().mockResolvedValue({
      commands: [{ name: 'review', description: 'Run review' }],
      agents: [],
    }),
    ...overrides,
  };
}

describe('workflowKeys', () => {
  it('includes BACKEND_VERSION prefix', () => {
    expect(workflowKeys.all[0]).toBe(BACKEND_VERSION);
  });

  it('generates correct query keys', () => {
    expect(workflowKeys.all).toEqual([BACKEND_VERSION, 'workflows']);
    expect(workflowKeys.ootb('my-project')).toEqual([
      BACKEND_VERSION,
      'workflows',
      'ootb',
      'my-project',
    ]);
    expect(workflowKeys.ootb()).toEqual([
      BACKEND_VERSION,
      'workflows',
      'ootb',
      undefined,
    ]);
    expect(workflowKeys.metadata('proj', 'sess-1')).toEqual([
      BACKEND_VERSION,
      'workflows',
      'metadata',
      'proj',
      'sess-1',
    ]);
  });
});

describe('useOOTBWorkflows', () => {
  it('calls port.listOOTBWorkflows and returns data', async () => {
    const fakePort = createFakeWorkflowsPort();
    const { result } = renderHook(
      () => useOOTBWorkflows('my-project', fakePort),
      { wrapper: createWrapper() },
    );

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.listOOTBWorkflows).toHaveBeenCalledWith('my-project');
    expect(result.current.data).toHaveLength(1);
    expect(result.current.data?.[0].name).toBe('code-review');
  });

  it('uses the correct query key', async () => {
    const queryClient = createTestQueryClient();
    const fakePort = createFakeWorkflowsPort();
    const { result } = renderHook(
      () => useOOTBWorkflows('my-project', fakePort),
      { wrapper: createWrapper(queryClient) },
    );

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    const cache = queryClient.getQueryCache().findAll();
    expect(cache).toHaveLength(1);
    expect(cache[0].queryKey).toEqual(workflowKeys.ootb('my-project'));
  });

  it('is disabled when projectName is empty', () => {
    const fakePort = createFakeWorkflowsPort();
    const { result } = renderHook(
      () => useOOTBWorkflows('', fakePort),
      { wrapper: createWrapper() },
    );

    expect(result.current.fetchStatus).toBe('idle');
    expect(fakePort.listOOTBWorkflows).not.toHaveBeenCalled();
  });

  it('is disabled when projectName is undefined', () => {
    const fakePort = createFakeWorkflowsPort();
    const { result } = renderHook(
      () => useOOTBWorkflows(undefined, fakePort),
      { wrapper: createWrapper() },
    );

    expect(result.current.fetchStatus).toBe('idle');
    expect(fakePort.listOOTBWorkflows).not.toHaveBeenCalled();
  });
});

describe('useWorkflowMetadata', () => {
  it('calls port.getWorkflowMetadata and returns data', async () => {
    const fakePort = createFakeWorkflowsPort();
    const { result } = renderHook(
      () => useWorkflowMetadata('proj', 'sess-1', true, fakePort),
      { wrapper: createWrapper() },
    );

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.getWorkflowMetadata).toHaveBeenCalledWith('proj', 'sess-1');
    expect(result.current.data?.commands).toHaveLength(1);
  });

  it('uses the correct query key', async () => {
    const queryClient = createTestQueryClient();
    const fakePort = createFakeWorkflowsPort();
    const { result } = renderHook(
      () => useWorkflowMetadata('proj', 'sess-1', true, fakePort),
      { wrapper: createWrapper(queryClient) },
    );

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    const cache = queryClient.getQueryCache().findAll();
    expect(cache).toHaveLength(1);
    expect(cache[0].queryKey).toEqual(workflowKeys.metadata('proj', 'sess-1'));
  });

  it('is disabled when enabled is false', () => {
    const fakePort = createFakeWorkflowsPort();
    const { result } = renderHook(
      () => useWorkflowMetadata('proj', 'sess-1', false, fakePort),
      { wrapper: createWrapper() },
    );

    expect(result.current.fetchStatus).toBe('idle');
    expect(fakePort.getWorkflowMetadata).not.toHaveBeenCalled();
  });

  it('is disabled when projectName is empty', () => {
    const fakePort = createFakeWorkflowsPort();
    const { result } = renderHook(
      () => useWorkflowMetadata('', 'sess-1', true, fakePort),
      { wrapper: createWrapper() },
    );

    expect(result.current.fetchStatus).toBe('idle');
    expect(fakePort.getWorkflowMetadata).not.toHaveBeenCalled();
  });

  it('is disabled when sessionName is empty', () => {
    const fakePort = createFakeWorkflowsPort();
    const { result } = renderHook(
      () => useWorkflowMetadata('proj', '', true, fakePort),
      { wrapper: createWrapper() },
    );

    expect(result.current.fetchStatus).toBe('idle');
    expect(fakePort.getWorkflowMetadata).not.toHaveBeenCalled();
  });
});
