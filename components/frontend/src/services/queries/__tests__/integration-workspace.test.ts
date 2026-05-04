import { renderHook, waitFor, act } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { createSessionWorkspaceAdapter } from '../../adapters/session-workspace';
import {
  useWorkspaceList,
  useWorkspaceFile,
  useWriteWorkspaceFile,
  useSessionGitHubDiff,
  useAllSessionGitHubDiffs,
  usePushSessionToGitHub,
  useAbandonSessionChanges,
  useGitMergeStatus,
  useGitCreateBranch,
  useGitListBranches,
  useGitStatus,
  useConfigureGitRemote,
} from '../use-workspace';
import { createWrapper } from './test-utils';

function createFakeWorkspaceApi() {
  return {
    listWorkspace: vi.fn().mockResolvedValue([
      { name: 'src', type: 'directory' },
      { name: 'README.md', type: 'file' },
    ]),
    readWorkspaceFile: vi.fn().mockResolvedValue('# Hello\nFile content here'),
    writeWorkspaceFile: vi.fn().mockResolvedValue(undefined),
    getSessionGitHubDiff: vi.fn().mockResolvedValue({
      files: { added: 3, removed: 1 },
      total_added: 42,
      total_removed: 7,
    }),
    pushSessionToGitHub: vi.fn().mockResolvedValue(undefined),
    abandonSessionChanges: vi.fn().mockResolvedValue(undefined),
    getGitMergeStatus: vi.fn().mockResolvedValue({ canMergeClean: true, localChanges: 0, remoteCommitsAhead: 0, conflictingFiles: [], remoteBranchExists: true }),
    gitCreateBranch: vi.fn().mockResolvedValue(undefined),
    gitListBranches: vi.fn().mockResolvedValue(['main', 'feature/new-ui', 'fix/bug-123']),
    gitStatus: vi.fn().mockResolvedValue({
      branch: 'main',
      hasChanges: true,
      initialized: true,
    }),
    configureGitRemote: vi.fn().mockResolvedValue(undefined),
  };
}

describe('integration: hook → sessionWorkspaceAdapter → fakeApi', () => {
  it('useWorkspaceList: directory listing flows through', async () => {
    const fakeApi = createFakeWorkspaceApi();
    const adapter = createSessionWorkspaceAdapter(fakeApi);

    const { result } = renderHook(
      () => useWorkspaceList('proj', 'sess', '/src', undefined, adapter),
      { wrapper: createWrapper() },
    );

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.listWorkspace).toHaveBeenCalledWith('proj', 'sess', '/src');
    expect(result.current.data).toHaveLength(2);
    expect(result.current.data?.[0].name).toBe('src');
  });

  it('useWorkspaceFile: file content flows through', async () => {
    const fakeApi = createFakeWorkspaceApi();
    const adapter = createSessionWorkspaceAdapter(fakeApi);

    const { result } = renderHook(
      () => useWorkspaceFile('proj', 'sess', 'README.md', undefined, adapter),
      { wrapper: createWrapper() },
    );

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.readWorkspaceFile).toHaveBeenCalledWith('proj', 'sess', 'README.md');
    expect(result.current.data).toContain('# Hello');
  });

  it('useWriteWorkspaceFile: write flows through', async () => {
    const fakeApi = createFakeWorkspaceApi();
    const adapter = createSessionWorkspaceAdapter(fakeApi);

    const { result } = renderHook(
      () => useWriteWorkspaceFile(adapter),
      { wrapper: createWrapper() },
    );

    act(() => {
      result.current.mutate({
        projectName: 'proj',
        sessionName: 'sess',
        path: 'README.md',
        content: 'updated content',
      });
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.writeWorkspaceFile).toHaveBeenCalledWith('proj', 'sess', 'README.md', 'updated content');
  });

  it('useSessionGitHubDiff: diff data flows through', async () => {
    const fakeApi = createFakeWorkspaceApi();
    const adapter = createSessionWorkspaceAdapter(fakeApi);

    const { result } = renderHook(
      () => useSessionGitHubDiff('proj', 'sess', 0, '/repos/myrepo', undefined, adapter),
      { wrapper: createWrapper() },
    );

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.getSessionGitHubDiff).toHaveBeenCalledWith('proj', 'sess', 0, '/repos/myrepo');
    expect(result.current.data?.total_added).toBe(42);
    expect(result.current.data?.total_removed).toBe(7);
  });

  it('usePushSessionToGitHub: push flows through', async () => {
    const fakeApi = createFakeWorkspaceApi();
    const adapter = createSessionWorkspaceAdapter(fakeApi);

    const { result } = renderHook(
      () => usePushSessionToGitHub(adapter),
      { wrapper: createWrapper() },
    );

    act(() => {
      result.current.mutate({
        projectName: 'proj',
        sessionName: 'sess',
        repoIndex: 0,
        repoPath: '/repos/myrepo',
      });
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.pushSessionToGitHub).toHaveBeenCalledWith('proj', 'sess', 0, '/repos/myrepo');
  });

  it('useAbandonSessionChanges: abandon flows through', async () => {
    const fakeApi = createFakeWorkspaceApi();
    const adapter = createSessionWorkspaceAdapter(fakeApi);

    const { result } = renderHook(
      () => useAbandonSessionChanges(adapter),
      { wrapper: createWrapper() },
    );

    act(() => {
      result.current.mutate({
        projectName: 'proj',
        sessionName: 'sess',
        repoIndex: 0,
        repoPath: '/repos/myrepo',
      });
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.abandonSessionChanges).toHaveBeenCalledWith('proj', 'sess', 0, '/repos/myrepo');
  });

  it('useGitMergeStatus: merge status flows through', async () => {
    const fakeApi = createFakeWorkspaceApi();
    const adapter = createSessionWorkspaceAdapter(fakeApi);

    const { result } = renderHook(
      () => useGitMergeStatus('proj', 'sess', 'artifacts', 'main', true, adapter),
      { wrapper: createWrapper() },
    );

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.getGitMergeStatus).toHaveBeenCalledWith('proj', 'sess', 'artifacts', 'main');
    expect(result.current.data?.canMergeClean).toBe(true);
  });

  it('useGitCreateBranch: branch creation flows through', async () => {
    const fakeApi = createFakeWorkspaceApi();
    const adapter = createSessionWorkspaceAdapter(fakeApi);

    const { result } = renderHook(
      () => useGitCreateBranch(adapter),
      { wrapper: createWrapper() },
    );

    act(() => {
      result.current.mutate({
        projectName: 'proj',
        sessionName: 'sess',
        branchName: 'feature/new',
      });
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.gitCreateBranch).toHaveBeenCalledWith('proj', 'sess', 'feature/new', 'artifacts');
  });

  it('useGitListBranches: branch list flows through', async () => {
    const fakeApi = createFakeWorkspaceApi();
    const adapter = createSessionWorkspaceAdapter(fakeApi);

    const { result } = renderHook(
      () => useGitListBranches('proj', 'sess', 'artifacts', true, adapter),
      { wrapper: createWrapper() },
    );

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.gitListBranches).toHaveBeenCalledWith('proj', 'sess', 'artifacts');
    expect(result.current.data).toEqual(['main', 'feature/new-ui', 'fix/bug-123']);
  });

  it('useGitStatus: git status flows through', async () => {
    const fakeApi = createFakeWorkspaceApi();
    const adapter = createSessionWorkspaceAdapter(fakeApi);

    const { result } = renderHook(
      () => useGitStatus('proj', 'sess', '/workspace', undefined, adapter),
      { wrapper: createWrapper() },
    );

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.gitStatus).toHaveBeenCalledWith('proj', 'sess', '/workspace');
    expect(result.current.data?.hasChanges).toBe(true);
  });

  it('useConfigureGitRemote: remote configuration flows through', async () => {
    const fakeApi = createFakeWorkspaceApi();
    const adapter = createSessionWorkspaceAdapter(fakeApi);

    const { result } = renderHook(
      () => useConfigureGitRemote(adapter),
      { wrapper: createWrapper() },
    );

    act(() => {
      result.current.mutate({
        projectName: 'proj',
        sessionName: 'sess',
        path: '/workspace',
        remoteUrl: 'https://github.com/org/repo.git',
        branch: 'develop',
      });
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.configureGitRemote).toHaveBeenCalledWith('proj', 'sess', '/workspace', 'https://github.com/org/repo.git', 'develop');
  });

  it('useAllSessionGitHubDiffs: aggregated diffs flow through', async () => {
    const fakeApi = createFakeWorkspaceApi();
    const adapter = createSessionWorkspaceAdapter(fakeApi);

    const repos = [
      { input: { url: 'https://github.com/org/repo', branch: 'main' } },
    ];
    const deriveRepoFolder = (url: string) => url.split('/').pop() || '';

    const { result } = renderHook(
      () => useAllSessionGitHubDiffs('proj', 'sess', repos, deriveRepoFolder, undefined, adapter),
      { wrapper: createWrapper() },
    );

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.getSessionGitHubDiff).toHaveBeenCalled();
    expect(result.current.data?.[0]?.total_added).toBe(42);
  });
});
