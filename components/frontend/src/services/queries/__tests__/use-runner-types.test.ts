import { renderHook, waitFor } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { useRunnerTypes, runnerTypeKeys } from '../use-runner-types';
import type { RunnerTypesPort } from '../../ports/runner-types';
import { createWrapper, createTestQueryClient } from './test-utils';
import { BACKEND_VERSION } from '../query-keys';

const mockRunnerTypes = [
  {
    id: 'claude-agent-sdk',
    displayName: 'Claude Agent SDK',
    description: 'Claude Code agent runner',
    framework: 'claude-code',
    provider: 'anthropic',
    auth: {
      requiredSecretKeys: ['ANTHROPIC_API_KEY'],
      secretKeyLogic: 'all' as const,
      vertexSupported: false,
    },
  },
  {
    id: 'custom-runner',
    displayName: 'Custom Runner',
    description: 'A custom runner type',
    framework: 'custom',
    provider: 'self-hosted',
    auth: {
      requiredSecretKeys: [],
      secretKeyLogic: 'any' as const,
      vertexSupported: true,
    },
  },
];

function createFakeRunnerTypesPort(
  overrides?: Partial<RunnerTypesPort>,
): RunnerTypesPort {
  return {
    getRunnerTypes: vi.fn().mockResolvedValue(mockRunnerTypes),
    ...overrides,
  };
}

describe('runnerTypeKeys', () => {
  it('includes BACKEND_VERSION prefix', () => {
    expect(runnerTypeKeys.all[0]).toBe(BACKEND_VERSION);
    expect(runnerTypeKeys.all).toEqual([BACKEND_VERSION, 'runner-types']);
  });

  it('global key extends all', () => {
    expect(runnerTypeKeys.global()).toEqual([
      BACKEND_VERSION,
      'runner-types',
      'global',
    ]);
  });

  it('forProject key includes project name', () => {
    expect(runnerTypeKeys.forProject('my-project')).toEqual([
      BACKEND_VERSION,
      'runner-types',
      'my-project',
    ]);
  });
});

describe('useRunnerTypes', () => {
  it('calls port.getRunnerTypes and returns data', async () => {
    const fakePort = createFakeRunnerTypesPort();
    const { result } = renderHook(
      () => useRunnerTypes('my-project', fakePort),
      { wrapper: createWrapper() },
    );

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.getRunnerTypes).toHaveBeenCalledWith('my-project');
    expect(result.current.data).toEqual(mockRunnerTypes);
  });

  it('is disabled when projectName is empty', () => {
    const fakePort = createFakeRunnerTypesPort();
    const { result } = renderHook(() => useRunnerTypes('', fakePort), {
      wrapper: createWrapper(),
    });

    expect(result.current.fetchStatus).toBe('idle');
    expect(fakePort.getRunnerTypes).not.toHaveBeenCalled();
  });

  it('uses the correct cache key', async () => {
    const queryClient = createTestQueryClient();
    const fakePort = createFakeRunnerTypesPort();
    const { result } = renderHook(
      () => useRunnerTypes('my-project', fakePort),
      { wrapper: createWrapper(queryClient) },
    );

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    const cache = queryClient.getQueryCache().findAll();
    expect(cache).toHaveLength(1);
    expect(cache[0].queryKey).toEqual(
      runnerTypeKeys.forProject('my-project'),
    );
  });

  it('propagates errors from the port', async () => {
    const fakePort = createFakeRunnerTypesPort({
      getRunnerTypes: vi
        .fn()
        .mockRejectedValue(new Error('Runner types unavailable')),
    });
    const { result } = renderHook(
      () => useRunnerTypes('my-project', fakePort),
      { wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(result.current.error).toBeInstanceOf(Error);
  });
});
