import { renderHook, waitFor, act } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';

import { createTestQueryClient, createWrapper } from './test-utils';

// --- Key factories ---
import { coderabbitKeys } from '../use-coderabbit';
import { integrationsKeys } from '../use-integrations';
import { gerritKeys } from '../use-gerrit';
import { githubKeys } from '../use-github';
import { mcpCredentialsKeys } from '../use-mcp-credentials';
import { workspaceKeys } from '../use-workspace';
import { sessionKeys } from '../use-sessions';
import { featureFlagKeys } from '../use-feature-flags-admin';
import { jiraKeys } from '../use-jira';

// --- Mutation hooks ---
import { useConnectCodeRabbit, useDisconnectCodeRabbit } from '../use-coderabbit';
import { useConnectGerrit, useDisconnectGerrit } from '../use-gerrit';
import {
  useConnectGitHub,
  useDisconnectGitHub,
  useSaveGitHubPAT,
  useDeleteGitHubPAT,
} from '../use-github';
import { useConnectMCPServer, useDisconnectMCPServer } from '../use-mcp-credentials';
import { usePushSessionToGitHub } from '../use-workspace';
import { useToggleFeatureFlag } from '../use-feature-flags-admin';

// --- Port types ---
import type { CodeRabbitPort } from '../../ports/coderabbit';
import type { GerritPort } from '../../ports/gerrit';
import type { GitHubPort } from '../../ports/github';
import type { McpCredentialsPort } from '../../ports/mcp-credentials';
import type { SessionWorkspacePort } from '../../ports/session-workspace';
import type { FeatureFlagsPort } from '../../ports/feature-flags';

// ---------------------------------------------------------------------------
// Fake factory helpers
// ---------------------------------------------------------------------------

function createFakeCodeRabbitPort(): CodeRabbitPort {
  return {
    getCodeRabbitStatus: vi.fn().mockResolvedValue({ connected: true }),
    connectCodeRabbit: vi.fn().mockResolvedValue(undefined),
    disconnectCodeRabbit: vi.fn().mockResolvedValue(undefined),
  };
}

function createFakeGerritPort(): GerritPort {
  return {
    getGerritInstances: vi.fn().mockResolvedValue({ instances: [] }),
    getGerritInstanceStatus: vi.fn().mockResolvedValue({ connected: false }),
    connectGerrit: vi.fn().mockResolvedValue(undefined),
    disconnectGerrit: vi.fn().mockResolvedValue(undefined),
    testGerritConnection: vi.fn().mockResolvedValue({ success: true }),
  };
}

function createFakeGitHubPort(): GitHubPort {
  return {
    getGitHubStatus: vi.fn().mockResolvedValue({ connected: true }),
    connectGitHub: vi.fn().mockResolvedValue('connected'),
    disconnectGitHub: vi.fn().mockResolvedValue('disconnected'),
    listGitHubForks: vi.fn().mockResolvedValue([]),
    createGitHubFork: vi.fn().mockResolvedValue({ full_name: 'org/repo' }),
    getPRDiff: vi.fn().mockResolvedValue({ files: [] }),
    createPullRequest: vi.fn().mockResolvedValue({ url: 'https://github.com/pr/1', number: 1 }),
    saveGitHubPAT: vi.fn().mockResolvedValue(undefined),
    getGitHubPATStatus: vi.fn().mockResolvedValue({ configured: false }),
    deleteGitHubPAT: vi.fn().mockResolvedValue(undefined),
  };
}

function createFakeMcpCredentialsPort(): McpCredentialsPort {
  return {
    getMCPServerStatus: vi.fn().mockResolvedValue({ connected: false }),
    connectMCPServer: vi.fn().mockResolvedValue(undefined),
    disconnectMCPServer: vi.fn().mockResolvedValue(undefined),
  };
}

function createFakeSessionWorkspacePort(): SessionWorkspacePort {
  return {
    listWorkspace: vi.fn().mockResolvedValue([]),
    readFile: vi.fn().mockResolvedValue(''),
    writeFile: vi.fn().mockResolvedValue(undefined),
    getGitHubDiff: vi.fn().mockResolvedValue({
      files: { added: 0, removed: 0 },
      total_added: 0,
      total_removed: 0,
    }),
    pushToGitHub: vi.fn().mockResolvedValue(undefined),
    abandonChanges: vi.fn().mockResolvedValue(undefined),
    getGitMergeStatus: vi.fn().mockResolvedValue({ status: 'clean' }),
    gitCreateBranch: vi.fn().mockResolvedValue(undefined),
    gitListBranches: vi.fn().mockResolvedValue(['main']),
    gitStatus: vi.fn().mockResolvedValue({ clean: true }),
    configureGitRemote: vi.fn().mockResolvedValue(undefined),
  };
}

function createFakeFeatureFlagsPort(): FeatureFlagsPort {
  return {
    getFeatureFlags: vi.fn().mockResolvedValue([]),
    evaluateFeatureFlag: vi.fn().mockResolvedValue({
      flag: 'test-flag',
      enabled: true,
      source: 'default' as const,
    }),
    getFeatureFlag: vi.fn().mockResolvedValue({
      name: 'test-flag',
      enabled: true,
    }),
    setFeatureFlagOverride: vi.fn().mockResolvedValue({
      message: 'ok',
      flag: 'test-flag',
      enabled: true,
      source: 'workspace-override',
    }),
    removeFeatureFlagOverride: vi.fn().mockResolvedValue({
      message: 'ok',
      flag: 'test-flag',
      enabled: false,
      source: 'default',
    }),
    enableFeatureFlag: vi.fn().mockResolvedValue({
      message: 'ok',
      flag: 'test-flag',
      enabled: true,
      source: 'unleash',
    }),
    disableFeatureFlag: vi.fn().mockResolvedValue({
      message: 'ok',
      flag: 'test-flag',
      enabled: false,
      source: 'unleash',
    }),
  };
}

// ---------------------------------------------------------------------------
// 1. CodeRabbit → Integrations
// ---------------------------------------------------------------------------
describe('coderabbit → integrations cross-module invalidation', () => {
  it('useConnectCodeRabbit invalidates coderabbit status AND integrations status', async () => {
    const queryClient = createTestQueryClient();
    queryClient.setQueryData(coderabbitKeys.status(), { connected: false });
    queryClient.setQueryData(integrationsKeys.status(), { github: true });

    const fakePort = createFakeCodeRabbitPort();
    const { result } = renderHook(() => useConnectCodeRabbit(fakePort), {
      wrapper: createWrapper(queryClient),
    });

    act(() => {
      result.current.mutate({ apiKey: 'test-key' });
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(queryClient.getQueryState(coderabbitKeys.status())?.isInvalidated).toBe(true);
    expect(queryClient.getQueryState(integrationsKeys.status())?.isInvalidated).toBe(true);
  });

  it('useDisconnectCodeRabbit invalidates coderabbit status AND integrations status', async () => {
    const queryClient = createTestQueryClient();
    queryClient.setQueryData(coderabbitKeys.status(), { connected: true });
    queryClient.setQueryData(integrationsKeys.status(), { coderabbit: true });

    const fakePort = createFakeCodeRabbitPort();
    const { result } = renderHook(() => useDisconnectCodeRabbit(fakePort), {
      wrapper: createWrapper(queryClient),
    });

    act(() => {
      result.current.mutate();
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(queryClient.getQueryState(coderabbitKeys.status())?.isInvalidated).toBe(true);
    expect(queryClient.getQueryState(integrationsKeys.status())?.isInvalidated).toBe(true);
  });
});

// ---------------------------------------------------------------------------
// 2. Gerrit → Integrations
// ---------------------------------------------------------------------------
describe('gerrit → integrations cross-module invalidation', () => {
  it('useConnectGerrit invalidates gerrit instances AND integrations status', async () => {
    const queryClient = createTestQueryClient();
    queryClient.setQueryData(gerritKeys.instances(), { instances: [] });
    queryClient.setQueryData(integrationsKeys.status(), { gerrit: false });

    const fakePort = createFakeGerritPort();
    const { result } = renderHook(() => useConnectGerrit(fakePort), {
      wrapper: createWrapper(queryClient),
    });

    act(() => {
      result.current.mutate({
        instanceName: 'my-gerrit',
        url: 'https://gerrit.example.com',
        authMethod: 'http_basic' as const,
        username: 'user',
        httpToken: 'token-abc',
      });
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(queryClient.getQueryState(gerritKeys.instances())?.isInvalidated).toBe(true);
    expect(queryClient.getQueryState(integrationsKeys.status())?.isInvalidated).toBe(true);
  });

  it('useDisconnectGerrit invalidates gerrit instances AND integrations status', async () => {
    const queryClient = createTestQueryClient();
    queryClient.setQueryData(gerritKeys.instances(), { instances: [{ name: 'my-gerrit' }] });
    queryClient.setQueryData(integrationsKeys.status(), { gerrit: true });

    const fakePort = createFakeGerritPort();
    const { result } = renderHook(() => useDisconnectGerrit(fakePort), {
      wrapper: createWrapper(queryClient),
    });

    act(() => {
      result.current.mutate('my-gerrit');
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(queryClient.getQueryState(gerritKeys.instances())?.isInvalidated).toBe(true);
    expect(queryClient.getQueryState(integrationsKeys.status())?.isInvalidated).toBe(true);
  });
});

// ---------------------------------------------------------------------------
// 3. GitHub → Integrations
// ---------------------------------------------------------------------------
describe('github → integrations cross-module invalidation', () => {
  it('useConnectGitHub invalidates github status AND integrations status', async () => {
    const queryClient = createTestQueryClient();
    queryClient.setQueryData(githubKeys.status(), { connected: false });
    queryClient.setQueryData(integrationsKeys.status(), { github: false });

    const fakePort = createFakeGitHubPort();
    const { result } = renderHook(() => useConnectGitHub(fakePort), {
      wrapper: createWrapper(queryClient),
    });

    act(() => {
      result.current.mutate({ installationId: 12345 });
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(queryClient.getQueryState(githubKeys.status())?.isInvalidated).toBe(true);
    expect(queryClient.getQueryState(integrationsKeys.status())?.isInvalidated).toBe(true);
  });

  it('useDisconnectGitHub invalidates github status, integrations status, AND github forks', async () => {
    const queryClient = createTestQueryClient();
    queryClient.setQueryData(githubKeys.status(), { connected: true });
    queryClient.setQueryData(integrationsKeys.status(), { github: true });
    queryClient.setQueryData(githubKeys.forks(), [{ full_name: 'org/repo' }]);

    const fakePort = createFakeGitHubPort();
    const { result } = renderHook(() => useDisconnectGitHub(fakePort), {
      wrapper: createWrapper(queryClient),
    });

    act(() => {
      result.current.mutate();
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(queryClient.getQueryState(githubKeys.status())?.isInvalidated).toBe(true);
    expect(queryClient.getQueryState(integrationsKeys.status())?.isInvalidated).toBe(true);
    expect(queryClient.getQueryState(githubKeys.forks())?.isInvalidated).toBe(true);
  });

  it('useSaveGitHubPAT invalidates github status AND integrations status', async () => {
    const queryClient = createTestQueryClient();
    queryClient.setQueryData(githubKeys.status(), { connected: true });
    queryClient.setQueryData(integrationsKeys.status(), { github: true });

    const fakePort = createFakeGitHubPort();
    const { result } = renderHook(() => useSaveGitHubPAT(fakePort), {
      wrapper: createWrapper(queryClient),
    });

    act(() => {
      result.current.mutate('ghp_test-token-12345');
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(queryClient.getQueryState(githubKeys.status())?.isInvalidated).toBe(true);
    expect(queryClient.getQueryState(integrationsKeys.status())?.isInvalidated).toBe(true);
  });

  it('useDeleteGitHubPAT invalidates github status AND integrations status', async () => {
    const queryClient = createTestQueryClient();
    queryClient.setQueryData(githubKeys.status(), { connected: true });
    queryClient.setQueryData(integrationsKeys.status(), { github: true });

    const fakePort = createFakeGitHubPort();
    const { result } = renderHook(() => useDeleteGitHubPAT(fakePort), {
      wrapper: createWrapper(queryClient),
    });

    act(() => {
      result.current.mutate();
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(queryClient.getQueryState(githubKeys.status())?.isInvalidated).toBe(true);
    expect(queryClient.getQueryState(integrationsKeys.status())?.isInvalidated).toBe(true);
  });
});

// ---------------------------------------------------------------------------
// 4. MCP Credentials → Integrations
// ---------------------------------------------------------------------------
describe('mcp-credentials → integrations cross-module invalidation', () => {
  const SERVER_NAME = 'test-mcp-server';

  it('useConnectMCPServer invalidates mcp-credentials status AND integrations status', async () => {
    const queryClient = createTestQueryClient();
    queryClient.setQueryData(mcpCredentialsKeys.status(SERVER_NAME), { connected: false });
    queryClient.setQueryData(integrationsKeys.status(), { mcp: false });

    const fakePort = createFakeMcpCredentialsPort();
    const { result } = renderHook(() => useConnectMCPServer(SERVER_NAME, fakePort), {
      wrapper: createWrapper(queryClient),
    });

    act(() => {
      result.current.mutate({ fields: { apiKey: 'mcp-key-123' } });
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(
      queryClient.getQueryState(mcpCredentialsKeys.status(SERVER_NAME))?.isInvalidated,
    ).toBe(true);
    expect(queryClient.getQueryState(integrationsKeys.status())?.isInvalidated).toBe(true);
  });

  it('useDisconnectMCPServer invalidates mcp-credentials status AND integrations status', async () => {
    const queryClient = createTestQueryClient();
    queryClient.setQueryData(mcpCredentialsKeys.status(SERVER_NAME), { connected: true });
    queryClient.setQueryData(integrationsKeys.status(), { mcp: true });

    const fakePort = createFakeMcpCredentialsPort();
    const { result } = renderHook(() => useDisconnectMCPServer(SERVER_NAME, fakePort), {
      wrapper: createWrapper(queryClient),
    });

    act(() => {
      result.current.mutate();
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(
      queryClient.getQueryState(mcpCredentialsKeys.status(SERVER_NAME))?.isInvalidated,
    ).toBe(true);
    expect(queryClient.getQueryState(integrationsKeys.status())?.isInvalidated).toBe(true);
  });
});

// ---------------------------------------------------------------------------
// 5. Workspace → Sessions
// ---------------------------------------------------------------------------
describe('workspace → sessions cross-module invalidation', () => {
  it('usePushSessionToGitHub invalidates workspace diff AND session detail', async () => {
    const projectName = 'test-project';
    const sessionName = 'test-session';
    const repoIndex = 0;

    const queryClient = createTestQueryClient();
    queryClient.setQueryData(workspaceKeys.diff(projectName, sessionName, repoIndex), {
      files: { added: 3, removed: 1 },
      total_added: 10,
      total_removed: 2,
    });
    queryClient.setQueryData(sessionKeys.detail(projectName, sessionName), {
      metadata: { name: sessionName },
      status: { phase: 'Running' },
    });

    const fakePort = createFakeSessionWorkspacePort();
    const { result } = renderHook(() => usePushSessionToGitHub(fakePort), {
      wrapper: createWrapper(queryClient),
    });

    act(() => {
      result.current.mutate({
        projectName,
        sessionName,
        repoIndex,
        repoPath: `/sessions/${sessionName}/workspace/my-repo`,
      });
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(
      queryClient.getQueryState(workspaceKeys.diff(projectName, sessionName, repoIndex))
        ?.isInvalidated,
    ).toBe(true);
    expect(
      queryClient.getQueryState(sessionKeys.detail(projectName, sessionName))?.isInvalidated,
    ).toBe(true);
  });
});

// ---------------------------------------------------------------------------
// 6. Feature Flags (intra-module cross-key invalidation)
// ---------------------------------------------------------------------------
describe('feature-flags cross-key invalidation', () => {
  it('useToggleFeatureFlag invalidates flag list AND flag evaluation', async () => {
    const projectName = 'test-project';
    const flagName = 'dark-mode';

    const queryClient = createTestQueryClient();
    queryClient.setQueryData(featureFlagKeys.list(projectName), [
      { name: flagName, enabled: false },
    ]);
    queryClient.setQueryData(featureFlagKeys.evaluate(projectName, flagName), {
      flag: flagName,
      enabled: false,
      source: 'default',
    });

    const fakePort = createFakeFeatureFlagsPort();
    const { result } = renderHook(() => useToggleFeatureFlag(fakePort), {
      wrapper: createWrapper(queryClient),
    });

    act(() => {
      result.current.mutate({ projectName, flagName, enable: true });
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(
      queryClient.getQueryState(featureFlagKeys.list(projectName))?.isInvalidated,
    ).toBe(true);
    expect(
      queryClient.getQueryState(featureFlagKeys.evaluate(projectName, flagName))?.isInvalidated,
    ).toBe(true);
  });
});

// ---------------------------------------------------------------------------
// 7. Negative tests — unrelated caches must NOT be invalidated
// ---------------------------------------------------------------------------
describe('negative: unrelated caches are not invalidated', () => {
  it('coderabbit connect does NOT invalidate jira, gerrit, github, or session caches', async () => {
    const queryClient = createTestQueryClient();
    // Seed the caches that SHOULD be invalidated
    queryClient.setQueryData(coderabbitKeys.status(), { connected: false });
    queryClient.setQueryData(integrationsKeys.status(), { github: true });
    // Seed unrelated caches
    queryClient.setQueryData(jiraKeys.status(), { connected: true });
    queryClient.setQueryData(gerritKeys.instances(), { instances: [] });
    queryClient.setQueryData(githubKeys.status(), { connected: true });
    queryClient.setQueryData(sessionKeys.detail('proj', 'sess'), {
      metadata: { name: 'sess' },
    });
    queryClient.setQueryData(featureFlagKeys.list('proj'), []);

    const fakePort = createFakeCodeRabbitPort();
    const { result } = renderHook(() => useConnectCodeRabbit(fakePort), {
      wrapper: createWrapper(queryClient),
    });

    act(() => {
      result.current.mutate({ apiKey: 'test-key' });
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    // Target caches ARE invalidated
    expect(queryClient.getQueryState(coderabbitKeys.status())?.isInvalidated).toBe(true);
    expect(queryClient.getQueryState(integrationsKeys.status())?.isInvalidated).toBe(true);

    // Unrelated caches are NOT invalidated
    expect(queryClient.getQueryState(jiraKeys.status())?.isInvalidated).not.toBe(true);
    expect(queryClient.getQueryState(gerritKeys.instances())?.isInvalidated).not.toBe(true);
    expect(queryClient.getQueryState(githubKeys.status())?.isInvalidated).not.toBe(true);
    expect(
      queryClient.getQueryState(sessionKeys.detail('proj', 'sess'))?.isInvalidated,
    ).not.toBe(true);
    expect(
      queryClient.getQueryState(featureFlagKeys.list('proj'))?.isInvalidated,
    ).not.toBe(true);
  });

  it('gerrit connect does NOT invalidate coderabbit, jira, or github caches', async () => {
    const queryClient = createTestQueryClient();
    queryClient.setQueryData(gerritKeys.instances(), { instances: [] });
    queryClient.setQueryData(integrationsKeys.status(), { gerrit: false });
    // Unrelated
    queryClient.setQueryData(coderabbitKeys.status(), { connected: true });
    queryClient.setQueryData(jiraKeys.status(), { connected: true });
    queryClient.setQueryData(githubKeys.status(), { connected: true });

    const fakePort = createFakeGerritPort();
    const { result } = renderHook(() => useConnectGerrit(fakePort), {
      wrapper: createWrapper(queryClient),
    });

    act(() => {
      result.current.mutate({
        instanceName: 'my-gerrit',
        url: 'https://gerrit.example.com',
        authMethod: 'http_basic' as const,
        username: 'user',
        httpToken: 'token-abc',
      });
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    // Target caches ARE invalidated
    expect(queryClient.getQueryState(gerritKeys.instances())?.isInvalidated).toBe(true);
    expect(queryClient.getQueryState(integrationsKeys.status())?.isInvalidated).toBe(true);

    // Unrelated caches are NOT invalidated
    expect(queryClient.getQueryState(coderabbitKeys.status())?.isInvalidated).not.toBe(true);
    expect(queryClient.getQueryState(jiraKeys.status())?.isInvalidated).not.toBe(true);
    expect(queryClient.getQueryState(githubKeys.status())?.isInvalidated).not.toBe(true);
  });

  it('workspace push does NOT invalidate integrations or feature-flag caches', async () => {
    const projectName = 'proj';
    const sessionName = 'sess';
    const repoIndex = 0;

    const queryClient = createTestQueryClient();
    queryClient.setQueryData(workspaceKeys.diff(projectName, sessionName, repoIndex), {
      files: { added: 1, removed: 0 },
      total_added: 5,
      total_removed: 0,
    });
    queryClient.setQueryData(sessionKeys.detail(projectName, sessionName), {
      metadata: { name: sessionName },
    });
    // Unrelated
    queryClient.setQueryData(integrationsKeys.status(), { github: true });
    queryClient.setQueryData(featureFlagKeys.list(projectName), []);
    queryClient.setQueryData(coderabbitKeys.status(), { connected: true });

    const fakePort = createFakeSessionWorkspacePort();
    const { result } = renderHook(() => usePushSessionToGitHub(fakePort), {
      wrapper: createWrapper(queryClient),
    });

    act(() => {
      result.current.mutate({
        projectName,
        sessionName,
        repoIndex,
        repoPath: `/sessions/${sessionName}/workspace/repo`,
      });
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    // Target caches ARE invalidated
    expect(
      queryClient.getQueryState(workspaceKeys.diff(projectName, sessionName, repoIndex))
        ?.isInvalidated,
    ).toBe(true);
    expect(
      queryClient.getQueryState(sessionKeys.detail(projectName, sessionName))?.isInvalidated,
    ).toBe(true);

    // Unrelated caches are NOT invalidated
    expect(queryClient.getQueryState(integrationsKeys.status())?.isInvalidated).not.toBe(true);
    expect(
      queryClient.getQueryState(featureFlagKeys.list(projectName))?.isInvalidated,
    ).not.toBe(true);
    expect(queryClient.getQueryState(coderabbitKeys.status())?.isInvalidated).not.toBe(true);
  });

  it('feature flag toggle does NOT invalidate integrations, sessions, or other domain caches', async () => {
    const projectName = 'proj';
    const flagName = 'my-flag';

    const queryClient = createTestQueryClient();
    queryClient.setQueryData(featureFlagKeys.list(projectName), []);
    queryClient.setQueryData(featureFlagKeys.evaluate(projectName, flagName), {
      flag: flagName,
      enabled: false,
      source: 'default',
    });
    // Unrelated
    queryClient.setQueryData(integrationsKeys.status(), { github: true });
    queryClient.setQueryData(sessionKeys.detail(projectName, 'sess'), {
      metadata: { name: 'sess' },
    });
    queryClient.setQueryData(jiraKeys.status(), { connected: true });

    const fakePort = createFakeFeatureFlagsPort();
    const { result } = renderHook(() => useToggleFeatureFlag(fakePort), {
      wrapper: createWrapper(queryClient),
    });

    act(() => {
      result.current.mutate({ projectName, flagName, enable: true });
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    // Target caches ARE invalidated
    expect(
      queryClient.getQueryState(featureFlagKeys.list(projectName))?.isInvalidated,
    ).toBe(true);
    expect(
      queryClient.getQueryState(featureFlagKeys.evaluate(projectName, flagName))?.isInvalidated,
    ).toBe(true);

    // Unrelated caches are NOT invalidated
    expect(queryClient.getQueryState(integrationsKeys.status())?.isInvalidated).not.toBe(true);
    expect(
      queryClient.getQueryState(sessionKeys.detail(projectName, 'sess'))?.isInvalidated,
    ).not.toBe(true);
    expect(queryClient.getQueryState(jiraKeys.status())?.isInvalidated).not.toBe(true);
  });
});
