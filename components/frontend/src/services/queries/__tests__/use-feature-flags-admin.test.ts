import { renderHook, waitFor, act } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import {
  featureFlagKeys,
  useFeatureFlags,
  useWorkspaceFlag,
  useFeatureFlag,
  useToggleFeatureFlag,
  useSetFeatureFlagOverride,
  useRemoveFeatureFlagOverride,
  useEnableFeatureFlag,
  useDisableFeatureFlag,
} from '../use-feature-flags-admin';
import type { FeatureFlagsPort } from '../../ports/feature-flags';
import { createWrapper, createTestQueryClient } from './test-utils';
import { BACKEND_VERSION } from '../query-keys';

const mockToggle = { name: 'test-flag', enabled: true, stale: false };
const mockToggles = [mockToggle, { name: 'other-flag', enabled: false, stale: false }];
const mockEvaluateResponse = { flag: 'test-flag', enabled: true, source: 'unleash' as const };
const mockToggleResponse = { message: 'ok', flag: 'test-flag', enabled: true, source: 'workspace-override' };

function createFakeFeatureFlagsPort(overrides?: Partial<FeatureFlagsPort>): FeatureFlagsPort {
  return {
    getFeatureFlags: vi.fn().mockResolvedValue(mockToggles),
    evaluateFeatureFlag: vi.fn().mockResolvedValue(mockEvaluateResponse),
    getFeatureFlag: vi.fn().mockResolvedValue(mockToggle),
    setFeatureFlagOverride: vi.fn().mockResolvedValue(mockToggleResponse),
    removeFeatureFlagOverride: vi.fn().mockResolvedValue(mockToggleResponse),
    enableFeatureFlag: vi.fn().mockResolvedValue(mockToggleResponse),
    disableFeatureFlag: vi.fn().mockResolvedValue(mockToggleResponse),
    ...overrides,
  };
}

describe('featureFlagKeys', () => {
  it('includes BACKEND_VERSION prefix in all keys', () => {
    expect(featureFlagKeys.all[0]).toBe(BACKEND_VERSION);
    expect(featureFlagKeys.all).toEqual([BACKEND_VERSION, 'feature-flags']);
  });

  it('generates correct list key', () => {
    expect(featureFlagKeys.list('my-project')).toEqual([BACKEND_VERSION, 'feature-flags', 'list', 'my-project']);
  });

  it('generates correct detail key', () => {
    expect(featureFlagKeys.detail('my-project', 'my-flag')).toEqual([
      BACKEND_VERSION, 'feature-flags', 'detail', 'my-project', 'my-flag',
    ]);
  });

  it('generates correct evaluate key', () => {
    expect(featureFlagKeys.evaluate('my-project', 'my-flag')).toEqual([
      BACKEND_VERSION, 'feature-flags', 'evaluate', 'my-project', 'my-flag',
    ]);
  });
});

describe('useFeatureFlags', () => {
  it('calls port.getFeatureFlags and returns data', async () => {
    const fakePort = createFakeFeatureFlagsPort();
    const { result } = renderHook(() => useFeatureFlags('my-project', fakePort), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.getFeatureFlags).toHaveBeenCalledWith('my-project');
    expect(result.current.data).toEqual(mockToggles);
  });

  it('is disabled when projectName is empty', () => {
    const fakePort = createFakeFeatureFlagsPort();
    const { result } = renderHook(() => useFeatureFlags('', fakePort), {
      wrapper: createWrapper(),
    });

    expect(result.current.fetchStatus).toBe('idle');
    expect(fakePort.getFeatureFlags).not.toHaveBeenCalled();
  });

  it('uses the correct query key', async () => {
    const queryClient = createTestQueryClient();
    const fakePort = createFakeFeatureFlagsPort();
    const { result } = renderHook(() => useFeatureFlags('proj', fakePort), {
      wrapper: createWrapper(queryClient),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    const cache = queryClient.getQueryCache().findAll();
    expect(cache[0].queryKey).toEqual(featureFlagKeys.list('proj'));
  });
});

describe('useWorkspaceFlag', () => {
  it('calls port.evaluateFeatureFlag and returns shaped data', async () => {
    const fakePort = createFakeFeatureFlagsPort();
    const { result } = renderHook(() => useWorkspaceFlag('proj', 'my-flag', fakePort), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isLoading).toBe(false));
    expect(fakePort.evaluateFeatureFlag).toHaveBeenCalledWith('proj', 'my-flag');
    expect(result.current.enabled).toBe(true);
    expect(result.current.source).toBe('unleash');
    expect(result.current.error).toBeNull();
  });

  it('returns enabled=false when data is not yet loaded', () => {
    const fakePort = createFakeFeatureFlagsPort({
      evaluateFeatureFlag: vi.fn().mockReturnValue(new Promise(() => {})),
    });
    const { result } = renderHook(() => useWorkspaceFlag('proj', 'my-flag', fakePort), {
      wrapper: createWrapper(),
    });

    expect(result.current.enabled).toBe(false);
    expect(result.current.source).toBeUndefined();
    expect(result.current.isLoading).toBe(true);
  });

  it('is disabled when projectName is empty', () => {
    const fakePort = createFakeFeatureFlagsPort();
    const { result } = renderHook(() => useWorkspaceFlag('', 'my-flag', fakePort), {
      wrapper: createWrapper(),
    });

    expect(result.current.isLoading).toBe(false);
    expect(fakePort.evaluateFeatureFlag).not.toHaveBeenCalled();
  });

  it('is disabled when flagName is empty', () => {
    const fakePort = createFakeFeatureFlagsPort();
    const { result } = renderHook(() => useWorkspaceFlag('proj', '', fakePort), {
      wrapper: createWrapper(),
    });

    expect(result.current.isLoading).toBe(false);
    expect(fakePort.evaluateFeatureFlag).not.toHaveBeenCalled();
  });
});

describe('useFeatureFlag', () => {
  it('calls port.getFeatureFlag and returns data', async () => {
    const fakePort = createFakeFeatureFlagsPort();
    const { result } = renderHook(() => useFeatureFlag('proj', 'my-flag', fakePort), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.getFeatureFlag).toHaveBeenCalledWith('proj', 'my-flag');
    expect(result.current.data).toEqual(mockToggle);
  });

  it('is disabled when projectName is empty', () => {
    const fakePort = createFakeFeatureFlagsPort();
    const { result } = renderHook(() => useFeatureFlag('', 'my-flag', fakePort), {
      wrapper: createWrapper(),
    });

    expect(result.current.fetchStatus).toBe('idle');
    expect(fakePort.getFeatureFlag).not.toHaveBeenCalled();
  });

  it('is disabled when flagName is empty', () => {
    const fakePort = createFakeFeatureFlagsPort();
    const { result } = renderHook(() => useFeatureFlag('proj', '', fakePort), {
      wrapper: createWrapper(),
    });

    expect(result.current.fetchStatus).toBe('idle');
    expect(fakePort.getFeatureFlag).not.toHaveBeenCalled();
  });
});

describe('useToggleFeatureFlag', () => {
  it('calls enableFeatureFlag when enable is true', async () => {
    const fakePort = createFakeFeatureFlagsPort();
    const { result } = renderHook(() => useToggleFeatureFlag(fakePort), {
      wrapper: createWrapper(),
    });

    act(() => {
      result.current.mutate({ projectName: 'proj', flagName: 'my-flag', enable: true });
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.enableFeatureFlag).toHaveBeenCalledWith('proj', 'my-flag');
    expect(fakePort.disableFeatureFlag).not.toHaveBeenCalled();
  });

  it('calls disableFeatureFlag when enable is false', async () => {
    const fakePort = createFakeFeatureFlagsPort();
    const { result } = renderHook(() => useToggleFeatureFlag(fakePort), {
      wrapper: createWrapper(),
    });

    act(() => {
      result.current.mutate({ projectName: 'proj', flagName: 'my-flag', enable: false });
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.disableFeatureFlag).toHaveBeenCalledWith('proj', 'my-flag');
    expect(fakePort.enableFeatureFlag).not.toHaveBeenCalled();
  });

  it('invalidates list and evaluate caches on success', async () => {
    const queryClient = createTestQueryClient();
    const fakePort = createFakeFeatureFlagsPort();

    const listKey = featureFlagKeys.list('proj');
    const evaluateKey = featureFlagKeys.evaluate('proj', 'my-flag');
    queryClient.setQueryData(listKey, mockToggles);
    queryClient.setQueryData(evaluateKey, mockEvaluateResponse);

    const { result } = renderHook(() => useToggleFeatureFlag(fakePort), {
      wrapper: createWrapper(queryClient),
    });

    act(() => {
      result.current.mutate({ projectName: 'proj', flagName: 'my-flag', enable: true });
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(queryClient.getQueryState(listKey)?.isInvalidated).toBe(true);
    expect(queryClient.getQueryState(evaluateKey)?.isInvalidated).toBe(true);
  });
});

describe('useSetFeatureFlagOverride', () => {
  it('calls port.setFeatureFlagOverride with correct args', async () => {
    const fakePort = createFakeFeatureFlagsPort();
    const { result } = renderHook(() => useSetFeatureFlagOverride(fakePort), {
      wrapper: createWrapper(),
    });

    act(() => {
      result.current.mutate({ projectName: 'proj', flagName: 'my-flag', enabled: true });
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.setFeatureFlagOverride).toHaveBeenCalledWith('proj', 'my-flag', true);
  });

  it('invalidates list and evaluate caches on success', async () => {
    const queryClient = createTestQueryClient();
    const fakePort = createFakeFeatureFlagsPort();

    const listKey = featureFlagKeys.list('proj');
    const evaluateKey = featureFlagKeys.evaluate('proj', 'my-flag');
    queryClient.setQueryData(listKey, mockToggles);
    queryClient.setQueryData(evaluateKey, mockEvaluateResponse);

    const { result } = renderHook(() => useSetFeatureFlagOverride(fakePort), {
      wrapper: createWrapper(queryClient),
    });

    act(() => {
      result.current.mutate({ projectName: 'proj', flagName: 'my-flag', enabled: false });
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(queryClient.getQueryState(listKey)?.isInvalidated).toBe(true);
    expect(queryClient.getQueryState(evaluateKey)?.isInvalidated).toBe(true);
  });
});

describe('useRemoveFeatureFlagOverride', () => {
  it('calls port.removeFeatureFlagOverride with correct args', async () => {
    const fakePort = createFakeFeatureFlagsPort();
    const { result } = renderHook(() => useRemoveFeatureFlagOverride(fakePort), {
      wrapper: createWrapper(),
    });

    act(() => {
      result.current.mutate({ projectName: 'proj', flagName: 'my-flag' });
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.removeFeatureFlagOverride).toHaveBeenCalledWith('proj', 'my-flag');
  });

  it('invalidates list and evaluate caches on success', async () => {
    const queryClient = createTestQueryClient();
    const fakePort = createFakeFeatureFlagsPort();

    const listKey = featureFlagKeys.list('proj');
    const evaluateKey = featureFlagKeys.evaluate('proj', 'my-flag');
    queryClient.setQueryData(listKey, mockToggles);
    queryClient.setQueryData(evaluateKey, mockEvaluateResponse);

    const { result } = renderHook(() => useRemoveFeatureFlagOverride(fakePort), {
      wrapper: createWrapper(queryClient),
    });

    act(() => {
      result.current.mutate({ projectName: 'proj', flagName: 'my-flag' });
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(queryClient.getQueryState(listKey)?.isInvalidated).toBe(true);
    expect(queryClient.getQueryState(evaluateKey)?.isInvalidated).toBe(true);
  });
});

describe('useEnableFeatureFlag', () => {
  it('calls port.enableFeatureFlag with correct args', async () => {
    const fakePort = createFakeFeatureFlagsPort();
    const { result } = renderHook(() => useEnableFeatureFlag(fakePort), {
      wrapper: createWrapper(),
    });

    act(() => {
      result.current.mutate({ projectName: 'proj', flagName: 'my-flag' });
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.enableFeatureFlag).toHaveBeenCalledWith('proj', 'my-flag');
  });

  it('invalidates list and evaluate caches on success', async () => {
    const queryClient = createTestQueryClient();
    const fakePort = createFakeFeatureFlagsPort();

    const listKey = featureFlagKeys.list('proj');
    const evaluateKey = featureFlagKeys.evaluate('proj', 'my-flag');
    queryClient.setQueryData(listKey, mockToggles);
    queryClient.setQueryData(evaluateKey, mockEvaluateResponse);

    const { result } = renderHook(() => useEnableFeatureFlag(fakePort), {
      wrapper: createWrapper(queryClient),
    });

    act(() => {
      result.current.mutate({ projectName: 'proj', flagName: 'my-flag' });
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(queryClient.getQueryState(listKey)?.isInvalidated).toBe(true);
    expect(queryClient.getQueryState(evaluateKey)?.isInvalidated).toBe(true);
  });
});

describe('useDisableFeatureFlag', () => {
  it('calls port.disableFeatureFlag with correct args', async () => {
    const fakePort = createFakeFeatureFlagsPort();
    const { result } = renderHook(() => useDisableFeatureFlag(fakePort), {
      wrapper: createWrapper(),
    });

    act(() => {
      result.current.mutate({ projectName: 'proj', flagName: 'my-flag' });
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.disableFeatureFlag).toHaveBeenCalledWith('proj', 'my-flag');
  });

  it('invalidates list and evaluate caches on success', async () => {
    const queryClient = createTestQueryClient();
    const fakePort = createFakeFeatureFlagsPort();

    const listKey = featureFlagKeys.list('proj');
    const evaluateKey = featureFlagKeys.evaluate('proj', 'my-flag');
    queryClient.setQueryData(listKey, mockToggles);
    queryClient.setQueryData(evaluateKey, mockEvaluateResponse);

    const { result } = renderHook(() => useDisableFeatureFlag(fakePort), {
      wrapper: createWrapper(queryClient),
    });

    act(() => {
      result.current.mutate({ projectName: 'proj', flagName: 'my-flag' });
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(queryClient.getQueryState(listKey)?.isInvalidated).toBe(true);
    expect(queryClient.getQueryState(evaluateKey)?.isInvalidated).toBe(true);
  });
});
