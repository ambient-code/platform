import { renderHook, waitFor, act } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { createAuthAdapter } from '../../adapters/auth';
import { createGitHubAdapter } from '../../adapters/github';
import { createGitLabAdapter } from '../../adapters/gitlab';
import { createGoogleAdapter } from '../../adapters/google';
import { createGerritAdapter } from '../../adapters/gerrit';
import { createJiraAdapter } from '../../adapters/jira';
import { createCodeRabbitAdapter } from '../../adapters/coderabbit';
import { createMcpCredentialsAdapter } from '../../adapters/mcp-credentials';
import { createIntegrationsAdapter } from '../../adapters/integrations';
import { useCurrentUser } from '../use-auth';
import {
  useGitHubStatus,
  useGitHubForks,
  usePRDiff,
  useConnectGitHub,
  useDisconnectGitHub,
  useCreateGitHubFork,
  useCreatePullRequest,
  useGitHubPATStatus,
  useSaveGitHubPAT,
  useDeleteGitHubPAT,
} from '../use-github';
import { useGitLabStatus, useConnectGitLab, useDisconnectGitLab } from '../use-gitlab';
import { useGoogleStatus, useDisconnectGoogle } from '../use-google';
import {
  useGerritInstances,
  useConnectGerrit,
  useDisconnectGerrit,
  useTestGerritConnection,
} from '../use-gerrit';
import { useJiraStatus, useConnectJira, useDisconnectJira } from '../use-jira';
import { useConnectCodeRabbit, useDisconnectCodeRabbit } from '../use-coderabbit';
import { useMCPServerStatus, useConnectMCPServer, useDisconnectMCPServer } from '../use-mcp-credentials';
import { useIntegrationsStatus } from '../use-integrations';
import { createWrapper } from './test-utils';

describe('integration: hook → authAdapter → fakeApi', () => {
  it('useCurrentUser: flows through', async () => {
    const fakeApi = { getCurrentUser: vi.fn().mockResolvedValue({ email: 'user@test.com', name: 'Test User' }) };
    const adapter = createAuthAdapter(fakeApi);

    const { result } = renderHook(() => useCurrentUser(adapter), { wrapper: createWrapper() });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.getCurrentUser).toHaveBeenCalled();
    expect(result.current.data?.email).toBe('user@test.com');
  });
});

describe('integration: hook → githubAdapter → fakeApi', () => {
  function createFakeGitHubApi() {
    return {
      getGitHubStatus: vi.fn().mockResolvedValue({ installed: true, pat: { configured: false } }),
      connectGitHub: vi.fn().mockResolvedValue('connected'),
      disconnectGitHub: vi.fn().mockResolvedValue('disconnected'),
      listGitHubForks: vi.fn().mockResolvedValue([{ full_name: 'user/repo', clone_url: 'https://github.com/user/repo.git' }]),
      createGitHubFork: vi.fn().mockResolvedValue({ name: 'forked', fullName: 'user/forked', owner: 'user', url: 'https://github.com/user/forked', defaultBranch: 'main' }),
      getPRDiff: vi.fn().mockResolvedValue({ additions: 10, deletions: 2, changedFiles: 3 }),
      createPullRequest: vi.fn().mockResolvedValue({ url: 'https://github.com/org/repo/pull/1', number: 1 }),
      saveGitHubPAT: vi.fn().mockResolvedValue(undefined),
      getGitHubPATStatus: vi.fn().mockResolvedValue({ configured: true, updatedAt: '2026-01-01' }),
      deleteGitHubPAT: vi.fn().mockResolvedValue(undefined),
    };
  }

  it('useGitHubStatus: flows through', async () => {
    const fakeApi = createFakeGitHubApi();
    const adapter = createGitHubAdapter(fakeApi);

    const { result } = renderHook(() => useGitHubStatus(adapter), { wrapper: createWrapper() });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.getGitHubStatus).toHaveBeenCalled();
    expect(result.current.data?.installed).toBe(true);
  });

  it('useGitHubForks: flows through', async () => {
    const fakeApi = createFakeGitHubApi();
    const adapter = createGitHubAdapter(fakeApi);

    const { result } = renderHook(
      () => useGitHubForks('proj', 'org/repo', adapter),
      { wrapper: createWrapper() },
    );

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.listGitHubForks).toHaveBeenCalledWith('proj', 'org/repo');
    expect(result.current.data).toHaveLength(1);
  });

  it('usePRDiff: flows through', async () => {
    const fakeApi = createFakeGitHubApi();
    const adapter = createGitHubAdapter(fakeApi);

    const { result } = renderHook(
      () => usePRDiff('owner', 'repo', 42, 'proj', adapter),
      { wrapper: createWrapper() },
    );

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.getPRDiff).toHaveBeenCalledWith('owner', 'repo', 42, 'proj');
    expect(result.current.data?.additions).toBe(10);
    expect(result.current.data?.deletions).toBe(2);
  });

  it('useConnectGitHub: flows through', async () => {
    const fakeApi = createFakeGitHubApi();
    const adapter = createGitHubAdapter(fakeApi);

    const { result } = renderHook(() => useConnectGitHub(adapter), { wrapper: createWrapper() });

    act(() => {
      result.current.mutate({ installationId: 123 });
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.connectGitHub).toHaveBeenCalledWith({ installationId: 123 });
  });

  it('useDisconnectGitHub: flows through', async () => {
    const fakeApi = createFakeGitHubApi();
    const adapter = createGitHubAdapter(fakeApi);

    const { result } = renderHook(() => useDisconnectGitHub(adapter), { wrapper: createWrapper() });

    act(() => { result.current.mutate(); });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.disconnectGitHub).toHaveBeenCalled();
  });

  it('useCreateGitHubFork: flows through', async () => {
    const fakeApi = createFakeGitHubApi();
    const adapter = createGitHubAdapter(fakeApi);

    const { result } = renderHook(() => useCreateGitHubFork(adapter), { wrapper: createWrapper() });

    act(() => {
      result.current.mutate({ data: { owner: 'org', repo: 'repo' }, projectName: 'proj' });
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.createGitHubFork).toHaveBeenCalledWith({ owner: 'org', repo: 'repo' }, 'proj');
    expect(result.current.data?.fullName).toBe('user/forked');
    expect(result.current.data?.defaultBranch).toBe('main');
  });

  it('useCreatePullRequest: flows through', async () => {
    const fakeApi = createFakeGitHubApi();
    const adapter = createGitHubAdapter(fakeApi);

    const { result } = renderHook(() => useCreatePullRequest(adapter), { wrapper: createWrapper() });

    act(() => {
      result.current.mutate({
        data: { owner: 'org', repo: 'repo', title: 'PR title', body: 'Description', head: 'feature', base: 'main' },
        projectName: 'proj',
      });
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.createPullRequest).toHaveBeenCalledWith(
      { owner: 'org', repo: 'repo', title: 'PR title', body: 'Description', head: 'feature', base: 'main' },
      'proj',
    );
    expect(result.current.data?.number).toBe(1);
    expect(result.current.data?.url).toBe('https://github.com/org/repo/pull/1');
  });

  it('useGitHubPATStatus: flows through', async () => {
    const fakeApi = createFakeGitHubApi();
    const adapter = createGitHubAdapter(fakeApi);

    const { result } = renderHook(() => useGitHubPATStatus(adapter), { wrapper: createWrapper() });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.getGitHubPATStatus).toHaveBeenCalled();
    expect(result.current.data?.configured).toBe(true);
  });

  it('useSaveGitHubPAT: flows through', async () => {
    const fakeApi = createFakeGitHubApi();
    const adapter = createGitHubAdapter(fakeApi);

    const { result } = renderHook(() => useSaveGitHubPAT(adapter), { wrapper: createWrapper() });

    act(() => { result.current.mutate('ghp_testtoken123'); });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.saveGitHubPAT).toHaveBeenCalledWith('ghp_testtoken123');
  });

  it('useDeleteGitHubPAT: flows through', async () => {
    const fakeApi = createFakeGitHubApi();
    const adapter = createGitHubAdapter(fakeApi);

    const { result } = renderHook(() => useDeleteGitHubPAT(adapter), { wrapper: createWrapper() });

    act(() => { result.current.mutate(); });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.deleteGitHubPAT).toHaveBeenCalled();
  });
});

describe('integration: hook → gitlabAdapter → fakeApi', () => {
  function createFakeGitLabApi() {
    return {
      getGitLabStatus: vi.fn().mockResolvedValue({ connected: true }),
      connectGitLab: vi.fn().mockResolvedValue(undefined),
      disconnectGitLab: vi.fn().mockResolvedValue(undefined),
    };
  }

  it('useGitLabStatus: flows through', async () => {
    const fakeApi = createFakeGitLabApi();
    const adapter = createGitLabAdapter(fakeApi);

    const { result } = renderHook(() => useGitLabStatus(adapter), { wrapper: createWrapper() });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.getGitLabStatus).toHaveBeenCalled();
    expect(result.current.data?.connected).toBe(true);
  });

  it('useConnectGitLab: flows through', async () => {
    const fakeApi = createFakeGitLabApi();
    const adapter = createGitLabAdapter(fakeApi);

    const { result } = renderHook(() => useConnectGitLab(adapter), { wrapper: createWrapper() });

    act(() => { result.current.mutate({ personalAccessToken: 'glpat-abc', instanceUrl: 'https://gitlab.example.com' }); });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.connectGitLab.mock.calls[0][0]).toEqual({ personalAccessToken: 'glpat-abc', instanceUrl: 'https://gitlab.example.com' });
  });

  it('useDisconnectGitLab: flows through', async () => {
    const fakeApi = createFakeGitLabApi();
    const adapter = createGitLabAdapter(fakeApi);

    const { result } = renderHook(() => useDisconnectGitLab(adapter), { wrapper: createWrapper() });

    act(() => { result.current.mutate(); });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.disconnectGitLab).toHaveBeenCalled();
  });
});

describe('integration: hook → googleAdapter → fakeApi', () => {
  function createFakeGoogleApi() {
    return {
      getGoogleOAuthURL: vi.fn().mockResolvedValue({ url: 'https://accounts.google.com/oauth' }),
      getGoogleStatus: vi.fn().mockResolvedValue({ connected: true, email: 'user@gmail.com' }),
      disconnectGoogle: vi.fn().mockResolvedValue(undefined),
    };
  }

  it('useGoogleStatus: flows through', async () => {
    const fakeApi = createFakeGoogleApi();
    const adapter = createGoogleAdapter(fakeApi);

    const { result } = renderHook(() => useGoogleStatus(adapter), { wrapper: createWrapper() });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.getGoogleStatus).toHaveBeenCalled();
    expect(result.current.data?.connected).toBe(true);
  });

  it('useDisconnectGoogle: flows through', async () => {
    const fakeApi = createFakeGoogleApi();
    const adapter = createGoogleAdapter(fakeApi);

    const { result } = renderHook(() => useDisconnectGoogle(adapter), { wrapper: createWrapper() });

    act(() => { result.current.mutate(); });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.disconnectGoogle).toHaveBeenCalled();
  });
});

describe('integration: hook → gerritAdapter → fakeApi', () => {
  function createFakeGerritApi() {
    return {
      getGerritInstances: vi.fn().mockResolvedValue({ instances: [{ name: 'gerrit-1', url: 'https://gerrit.example.com' }] }),
      getGerritInstanceStatus: vi.fn().mockResolvedValue({ connected: true }),
      connectGerrit: vi.fn().mockResolvedValue(undefined),
      disconnectGerrit: vi.fn().mockResolvedValue(undefined),
      testGerritConnection: vi.fn().mockResolvedValue({ valid: true, message: 'OK' }),
    };
  }

  it('useGerritInstances: flows through', async () => {
    const fakeApi = createFakeGerritApi();
    const adapter = createGerritAdapter(fakeApi);

    const { result } = renderHook(() => useGerritInstances(adapter), { wrapper: createWrapper() });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.getGerritInstances).toHaveBeenCalled();
    expect(result.current.data?.instances).toHaveLength(1);
  });

  it('useConnectGerrit: flows through', async () => {
    const fakeApi = createFakeGerritApi();
    const adapter = createGerritAdapter(fakeApi);

    const { result } = renderHook(() => useConnectGerrit(adapter), { wrapper: createWrapper() });

    act(() => { result.current.mutate({ instanceName: 'gerrit-1', url: 'https://gerrit.example.com', authMethod: 'http_basic' as const, username: 'user', httpToken: 'token' }); });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.connectGerrit.mock.calls[0][0]).toEqual({ instanceName: 'gerrit-1', url: 'https://gerrit.example.com', authMethod: 'http_basic', username: 'user', httpToken: 'token' });
  });

  it('useDisconnectGerrit: flows through', async () => {
    const fakeApi = createFakeGerritApi();
    const adapter = createGerritAdapter(fakeApi);

    const { result } = renderHook(() => useDisconnectGerrit(adapter), { wrapper: createWrapper() });

    act(() => { result.current.mutate('gerrit-1'); });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.disconnectGerrit).toHaveBeenCalled();
    expect(fakeApi.disconnectGerrit.mock.calls[0][0]).toBe('gerrit-1');
  });

  it('useTestGerritConnection: flows through', async () => {
    const fakeApi = createFakeGerritApi();
    const adapter = createGerritAdapter(fakeApi);

    const { result } = renderHook(() => useTestGerritConnection(adapter), { wrapper: createWrapper() });

    act(() => { result.current.mutate({ url: 'https://gerrit.example.com', authMethod: 'http_basic' as const, username: 'user', httpToken: 'token' }); });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.testGerritConnection.mock.calls[0][0]).toEqual({ url: 'https://gerrit.example.com', authMethod: 'http_basic', username: 'user', httpToken: 'token' });
    expect(result.current.data?.valid).toBe(true);
    expect(result.current.data?.message).toBe('OK');
  });
});

describe('integration: hook → jiraAdapter → fakeApi', () => {
  function createFakeJiraApi() {
    return {
      getJiraStatus: vi.fn().mockResolvedValue({ connected: true, url: 'https://jira.example.com' }),
      connectJira: vi.fn().mockResolvedValue(undefined),
      disconnectJira: vi.fn().mockResolvedValue(undefined),
    };
  }

  it('useJiraStatus: flows through', async () => {
    const fakeApi = createFakeJiraApi();
    const adapter = createJiraAdapter(fakeApi);

    const { result } = renderHook(() => useJiraStatus(adapter), { wrapper: createWrapper() });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.getJiraStatus).toHaveBeenCalled();
    expect(result.current.data?.connected).toBe(true);
  });

  it('useConnectJira: flows through', async () => {
    const fakeApi = createFakeJiraApi();
    const adapter = createJiraAdapter(fakeApi);

    const { result } = renderHook(() => useConnectJira(adapter), { wrapper: createWrapper() });

    act(() => { result.current.mutate({ url: 'https://jira.example.com', email: 'user@example.com', apiToken: 'jira-token' }); });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.connectJira.mock.calls[0][0]).toEqual({ url: 'https://jira.example.com', email: 'user@example.com', apiToken: 'jira-token' });
  });

  it('useDisconnectJira: flows through', async () => {
    const fakeApi = createFakeJiraApi();
    const adapter = createJiraAdapter(fakeApi);

    const { result } = renderHook(() => useDisconnectJira(adapter), { wrapper: createWrapper() });

    act(() => { result.current.mutate(); });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.disconnectJira).toHaveBeenCalled();
  });
});

describe('integration: hook → coderabbitAdapter → fakeApi', () => {
  function createFakeCodeRabbitApi() {
    return {
      getCodeRabbitStatus: vi.fn().mockResolvedValue({ connected: false }),
      connectCodeRabbit: vi.fn().mockResolvedValue(undefined),
      disconnectCodeRabbit: vi.fn().mockResolvedValue(undefined),
    };
  }

  it('useConnectCodeRabbit: flows through', async () => {
    const fakeApi = createFakeCodeRabbitApi();
    const adapter = createCodeRabbitAdapter(fakeApi);

    const { result } = renderHook(() => useConnectCodeRabbit(adapter), { wrapper: createWrapper() });

    act(() => { result.current.mutate({ apiKey: 'cr-key-123' }); });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.connectCodeRabbit.mock.calls[0][0]).toEqual({ apiKey: 'cr-key-123' });
  });

  it('useDisconnectCodeRabbit: flows through', async () => {
    const fakeApi = createFakeCodeRabbitApi();
    const adapter = createCodeRabbitAdapter(fakeApi);

    const { result } = renderHook(() => useDisconnectCodeRabbit(adapter), { wrapper: createWrapper() });

    act(() => { result.current.mutate(); });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.disconnectCodeRabbit).toHaveBeenCalled();
  });
});

describe('integration: hook → mcpCredentialsAdapter → fakeApi', () => {
  function createFakeMcpCredentialsApi() {
    return {
      getMCPServerStatus: vi.fn().mockResolvedValue({ connected: true, serverName: 'my-server' }),
      connectMCPServer: vi.fn().mockResolvedValue(undefined),
      disconnectMCPServer: vi.fn().mockResolvedValue(undefined),
    };
  }

  it('useMCPServerStatus: flows through', async () => {
    const fakeApi = createFakeMcpCredentialsApi();
    const adapter = createMcpCredentialsAdapter(fakeApi);

    const { result } = renderHook(
      () => useMCPServerStatus('my-server', adapter),
      { wrapper: createWrapper() },
    );

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.getMCPServerStatus).toHaveBeenCalledWith('my-server');
    expect(result.current.data?.connected).toBe(true);
  });

  it('useConnectMCPServer: flows through', async () => {
    const fakeApi = createFakeMcpCredentialsApi();
    const adapter = createMcpCredentialsAdapter(fakeApi);

    const { result } = renderHook(
      () => useConnectMCPServer('my-server', adapter),
      { wrapper: createWrapper() },
    );

    act(() => { result.current.mutate({ fields: { token: 'abc' } }); });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.connectMCPServer).toHaveBeenCalledWith('my-server', { fields: { token: 'abc' } });
  });

  it('useDisconnectMCPServer: flows through', async () => {
    const fakeApi = createFakeMcpCredentialsApi();
    const adapter = createMcpCredentialsAdapter(fakeApi);

    const { result } = renderHook(
      () => useDisconnectMCPServer('my-server', adapter),
      { wrapper: createWrapper() },
    );

    act(() => { result.current.mutate(); });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.disconnectMCPServer).toHaveBeenCalledWith('my-server');
  });
});

describe('integration: hook → integrationsAdapter → fakeApi', () => {
  it('useIntegrationsStatus: flows through', async () => {
    const fakeApi = { getIntegrationsStatus: vi.fn().mockResolvedValue({ github: { connected: true }, gitlab: { connected: false } }) };
    const adapter = createIntegrationsAdapter(fakeApi);

    const { result } = renderHook(() => useIntegrationsStatus(adapter), { wrapper: createWrapper() });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.getIntegrationsStatus).toHaveBeenCalled();
    expect(result.current.data).toEqual({ github: { connected: true }, gitlab: { connected: false } });
  });
});
