import { renderHook, waitFor } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import {
  repoKeys,
  useRepoBlob,
  useRepoTree,
  useRepoFileExists,
  useRepoBranches,
} from '../use-repo';
import type { RepoPort } from '../../ports/repo';
import { createWrapper, createTestQueryClient } from './test-utils';
import { BACKEND_VERSION } from '../query-keys';

const sampleParams = { repo: 'my-repo', ref: 'main', path: 'src/index.ts' };
const emptyParams = { repo: '', ref: '', path: '' };

function createFakeRepoPort(overrides?: Partial<RepoPort>): RepoPort {
  return {
    getRepoBlob: vi.fn().mockResolvedValue(new Response('file content')),
    getRepoTree: vi.fn().mockResolvedValue({
      entries: [
        { type: 'blob', name: 'index.ts', path: 'src/index.ts', sha: 'abc123' },
      ],
      sha: 'tree-sha',
    }),
    checkFileExists: vi.fn().mockResolvedValue(true),
    listRepoBranches: vi.fn().mockResolvedValue({
      branches: ['main', 'develop'],
    }),
    ...overrides,
  };
}

describe('repoKeys', () => {
  it('includes BACKEND_VERSION prefix', () => {
    expect(repoKeys.all[0]).toBe(BACKEND_VERSION);
  });

  it('generates correct query keys', () => {
    expect(repoKeys.all).toEqual([BACKEND_VERSION, 'repo']);
    expect(repoKeys.blob('proj', sampleParams)).toEqual([
      BACKEND_VERSION,
      'repo',
      'blob',
      'proj',
      'my-repo',
      'main',
      'src/index.ts',
    ]);
    expect(repoKeys.tree('proj', sampleParams)).toEqual([
      BACKEND_VERSION,
      'repo',
      'tree',
      'proj',
      'my-repo',
      'main',
      'src/index.ts',
    ]);
    expect(repoKeys.repoBranches('proj', 'my-repo')).toEqual([
      BACKEND_VERSION,
      'repo',
      'branches',
      'proj',
      'my-repo',
    ]);
  });
});

describe('useRepoBlob', () => {
  it('calls port.getRepoBlob and returns data', async () => {
    const fakePort = createFakeRepoPort();
    const { result } = renderHook(
      () => useRepoBlob('proj', sampleParams, undefined, fakePort),
      { wrapper: createWrapper() },
    );

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.getRepoBlob).toHaveBeenCalledWith('proj', sampleParams);
    expect(result.current.data).toBeInstanceOf(Response);
  });

  it('uses the correct query key', async () => {
    const queryClient = createTestQueryClient();
    const fakePort = createFakeRepoPort();
    const { result } = renderHook(
      () => useRepoBlob('proj', sampleParams, undefined, fakePort),
      { wrapper: createWrapper(queryClient) },
    );

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    const cache = queryClient.getQueryCache().findAll();
    expect(cache).toHaveLength(1);
    expect(cache[0].queryKey).toEqual(repoKeys.blob('proj', sampleParams));
  });

  it('is disabled when projectName is empty', () => {
    const fakePort = createFakeRepoPort();
    const { result } = renderHook(
      () => useRepoBlob('', sampleParams, undefined, fakePort),
      { wrapper: createWrapper() },
    );

    expect(result.current.fetchStatus).toBe('idle');
    expect(fakePort.getRepoBlob).not.toHaveBeenCalled();
  });

  it('is disabled when params have empty fields', () => {
    const fakePort = createFakeRepoPort();
    const { result } = renderHook(
      () => useRepoBlob('proj', emptyParams, undefined, fakePort),
      { wrapper: createWrapper() },
    );

    expect(result.current.fetchStatus).toBe('idle');
    expect(fakePort.getRepoBlob).not.toHaveBeenCalled();
  });

  it('is disabled when options.enabled is false', () => {
    const fakePort = createFakeRepoPort();
    const { result } = renderHook(
      () => useRepoBlob('proj', sampleParams, { enabled: false }, fakePort),
      { wrapper: createWrapper() },
    );

    expect(result.current.fetchStatus).toBe('idle');
    expect(fakePort.getRepoBlob).not.toHaveBeenCalled();
  });
});

describe('useRepoTree', () => {
  it('calls port.getRepoTree and returns data', async () => {
    const fakePort = createFakeRepoPort();
    const { result } = renderHook(
      () => useRepoTree('proj', sampleParams, undefined, fakePort),
      { wrapper: createWrapper() },
    );

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.getRepoTree).toHaveBeenCalledWith('proj', sampleParams);
    expect(result.current.data?.entries).toHaveLength(1);
  });

  it('is disabled when projectName is empty', () => {
    const fakePort = createFakeRepoPort();
    const { result } = renderHook(
      () => useRepoTree('', sampleParams, undefined, fakePort),
      { wrapper: createWrapper() },
    );

    expect(result.current.fetchStatus).toBe('idle');
    expect(fakePort.getRepoTree).not.toHaveBeenCalled();
  });

  it('is disabled when params have empty fields', () => {
    const fakePort = createFakeRepoPort();
    const { result } = renderHook(
      () => useRepoTree('proj', emptyParams, undefined, fakePort),
      { wrapper: createWrapper() },
    );

    expect(result.current.fetchStatus).toBe('idle');
    expect(fakePort.getRepoTree).not.toHaveBeenCalled();
  });

  it('is disabled when options.enabled is false', () => {
    const fakePort = createFakeRepoPort();
    const { result } = renderHook(
      () => useRepoTree('proj', sampleParams, { enabled: false }, fakePort),
      { wrapper: createWrapper() },
    );

    expect(result.current.fetchStatus).toBe('idle');
    expect(fakePort.getRepoTree).not.toHaveBeenCalled();
  });
});

describe('useRepoFileExists', () => {
  it('calls port.checkFileExists and returns data', async () => {
    const fakePort = createFakeRepoPort();
    const { result } = renderHook(
      () => useRepoFileExists('proj', sampleParams, undefined, fakePort),
      { wrapper: createWrapper() },
    );

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.checkFileExists).toHaveBeenCalledWith('proj', sampleParams);
    expect(result.current.data).toBe(true);
  });

  it('uses the correct query key with exists suffix', async () => {
    const queryClient = createTestQueryClient();
    const fakePort = createFakeRepoPort();
    const { result } = renderHook(
      () => useRepoFileExists('proj', sampleParams, undefined, fakePort),
      { wrapper: createWrapper(queryClient) },
    );

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    const cache = queryClient.getQueryCache().findAll();
    expect(cache).toHaveLength(1);
    expect(cache[0].queryKey).toEqual([
      ...repoKeys.blob('proj', sampleParams),
      'exists',
    ]);
  });

  it('is disabled when projectName is empty', () => {
    const fakePort = createFakeRepoPort();
    const { result } = renderHook(
      () => useRepoFileExists('', sampleParams, undefined, fakePort),
      { wrapper: createWrapper() },
    );

    expect(result.current.fetchStatus).toBe('idle');
    expect(fakePort.checkFileExists).not.toHaveBeenCalled();
  });

  it('is disabled when options.enabled is false', () => {
    const fakePort = createFakeRepoPort();
    const { result } = renderHook(
      () => useRepoFileExists('proj', sampleParams, { enabled: false }, fakePort),
      { wrapper: createWrapper() },
    );

    expect(result.current.fetchStatus).toBe('idle');
    expect(fakePort.checkFileExists).not.toHaveBeenCalled();
  });
});

describe('useRepoBranches', () => {
  it('calls port.listRepoBranches and returns data', async () => {
    const fakePort = createFakeRepoPort();
    const { result } = renderHook(
      () => useRepoBranches('proj', 'my-repo', undefined, fakePort),
      { wrapper: createWrapper() },
    );

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.listRepoBranches).toHaveBeenCalledWith('proj', 'my-repo');
    expect(result.current.data).toEqual({ branches: ['main', 'develop'] });
  });

  it('uses the correct query key', async () => {
    const queryClient = createTestQueryClient();
    const fakePort = createFakeRepoPort();
    const { result } = renderHook(
      () => useRepoBranches('proj', 'my-repo', undefined, fakePort),
      { wrapper: createWrapper(queryClient) },
    );

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    const cache = queryClient.getQueryCache().findAll();
    expect(cache).toHaveLength(1);
    expect(cache[0].queryKey).toEqual(repoKeys.repoBranches('proj', 'my-repo'));
  });

  it('is disabled when projectName is empty', () => {
    const fakePort = createFakeRepoPort();
    const { result } = renderHook(
      () => useRepoBranches('', 'my-repo', undefined, fakePort),
      { wrapper: createWrapper() },
    );

    expect(result.current.fetchStatus).toBe('idle');
    expect(fakePort.listRepoBranches).not.toHaveBeenCalled();
  });

  it('is disabled when repo is empty', () => {
    const fakePort = createFakeRepoPort();
    const { result } = renderHook(
      () => useRepoBranches('proj', '', undefined, fakePort),
      { wrapper: createWrapper() },
    );

    expect(result.current.fetchStatus).toBe('idle');
    expect(fakePort.listRepoBranches).not.toHaveBeenCalled();
  });

  it('is disabled when options.enabled is false', () => {
    const fakePort = createFakeRepoPort();
    const { result } = renderHook(
      () => useRepoBranches('proj', 'my-repo', { enabled: false }, fakePort),
      { wrapper: createWrapper() },
    );

    expect(result.current.fetchStatus).toBe('idle');
    expect(fakePort.listRepoBranches).not.toHaveBeenCalled();
  });
});
