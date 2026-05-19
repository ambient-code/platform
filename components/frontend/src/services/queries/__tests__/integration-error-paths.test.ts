import { renderHook, waitFor, act } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { createSessionsAdapter } from '../../adapters/sessions';
import { createProjectsAdapter } from '../../adapters/projects';
import { createProjectAccessAdapter } from '../../adapters/project-access';
import { createSessionTasksAdapter } from '../../adapters/session-tasks';
import { createGitHubAdapter } from '../../adapters/github';
import { createKeysAdapter } from '../../adapters/keys';
import { createSessionWorkspaceAdapter } from '../../adapters/session-workspace';
import { createScheduledSessionsAdapter } from '../../adapters/scheduled-sessions';
import { createSecretsAdapter } from '../../adapters/secrets';
import { createAuthAdapter } from '../../adapters/auth';
import {
  useSessionsPaginated,
  useSession,
  useCreateSession,
  useStopSession,
} from '../use-sessions';
import { useProjectsPaginated, useProject, useDeleteProject } from '../use-projects';
import { useProjectAccess } from '../use-project-access';
import { useCurrentUser } from '../use-auth';
import { useGitHubStatus, useConnectGitHub } from '../use-github';
import { useKeys, useCreateKey } from '../use-keys';
import { useWorkspaceList, useWriteWorkspaceFile } from '../use-workspace';
import { useScheduledSessions, useDeleteScheduledSession } from '../use-scheduled-sessions';
import { useSecretsList, useUpdateSecrets } from '../use-secrets';
import { createWrapper } from './test-utils';

const apiError = new Error('API request failed: 500 Internal Server Error');

describe('error paths: pagination adapters propagate errors through hooks', () => {
  it('useSessionsPaginated: API rejection surfaces as hook error', async () => {
    const fakeApi = {
      listSessionsPaginated: vi.fn().mockRejectedValue(apiError),
      getSession: vi.fn(),
      createSession: vi.fn(),
      stopSession: vi.fn(),
      startSession: vi.fn(),
      cloneSession: vi.fn(),
      deleteSession: vi.fn(),
      getSessionPodEvents: vi.fn(),
      updateSessionDisplayName: vi.fn(),
      getSessionExport: vi.fn(),
      switchSessionModel: vi.fn(),
      saveToGoogleDrive: vi.fn(),
      listSessions: vi.fn(),
      getReposStatus: vi.fn(),
      getMcpStatus: vi.fn(),
      updateSessionMcpServers: vi.fn(),
      getCapabilities: vi.fn(),
    };
    const adapter = createSessionsAdapter(fakeApi);

    const { result } = renderHook(
      () => useSessionsPaginated('proj', {}, adapter),
      { wrapper: createWrapper() },
    );

    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(result.current.error).toBe(apiError);
    expect(result.current.data).toBeUndefined();
  });

  it('useProjectsPaginated: API rejection surfaces as hook error', async () => {
    const fakeApi = {
      listProjectsPaginated: vi.fn().mockRejectedValue(apiError),
      getProject: vi.fn(),
      createProject: vi.fn(),
      updateProject: vi.fn(),
      deleteProject: vi.fn(),
      getProjectIntegrationStatus: vi.fn(),
      getProjectMcpServers: vi.fn(),
      updateProjectMcpServers: vi.fn(),
      listProjects: vi.fn(),
      getProjectAccess: vi.fn(),
      getProjectPermissions: vi.fn(),
      addProjectPermission: vi.fn(),
      removeProjectPermission: vi.fn(),
    };
    const adapter = createProjectsAdapter(fakeApi);

    const { result } = renderHook(
      () => useProjectsPaginated({}, adapter),
      { wrapper: createWrapper() },
    );

    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(result.current.error).toBe(apiError);
  });
});

describe('error paths: single-resource queries propagate errors', () => {
  it('useSession: API rejection surfaces as hook error', { timeout: 30000 }, async () => {
    const fakeApi = {
      listSessionsPaginated: vi.fn(),
      getSession: vi.fn().mockRejectedValue(apiError),
      createSession: vi.fn(),
      stopSession: vi.fn(),
      startSession: vi.fn(),
      cloneSession: vi.fn(),
      deleteSession: vi.fn(),
      getSessionPodEvents: vi.fn(),
      updateSessionDisplayName: vi.fn(),
      getSessionExport: vi.fn(),
      switchSessionModel: vi.fn(),
      saveToGoogleDrive: vi.fn(),
      listSessions: vi.fn(),
      getReposStatus: vi.fn(),
      getMcpStatus: vi.fn(),
      updateSessionMcpServers: vi.fn(),
      getCapabilities: vi.fn(),
    };
    const adapter = createSessionsAdapter(fakeApi);

    const { result } = renderHook(
      () => useSession('proj', 'sess-1', adapter),
      { wrapper: createWrapper() },
    );

    await waitFor(() => expect(result.current.isError).toBe(true), { timeout: 30000 });
    expect(result.current.error).toBe(apiError);
  });

  it('useProject: API rejection surfaces as hook error', async () => {
    const fakeApi = {
      listProjectsPaginated: vi.fn(),
      getProject: vi.fn().mockRejectedValue(apiError),
      createProject: vi.fn(),
      updateProject: vi.fn(),
      deleteProject: vi.fn(),
      getProjectIntegrationStatus: vi.fn(),
      getProjectMcpServers: vi.fn(),
      updateProjectMcpServers: vi.fn(),
      listProjects: vi.fn(),
      getProjectAccess: vi.fn(),
      getProjectPermissions: vi.fn(),
      addProjectPermission: vi.fn(),
      removeProjectPermission: vi.fn(),
    };
    const adapter = createProjectsAdapter(fakeApi);

    const { result } = renderHook(
      () => useProject('my-project', adapter),
      { wrapper: createWrapper() },
    );

    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(result.current.error).toBe(apiError);
  });
});

describe('error paths: passthrough query adapters propagate errors', () => {
  it('useCurrentUser: API rejection surfaces as hook error', async () => {
    const fakeApi = { getCurrentUser: vi.fn().mockRejectedValue(apiError) };
    const adapter = createAuthAdapter(fakeApi);

    const { result } = renderHook(() => useCurrentUser(adapter), { wrapper: createWrapper() });

    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(result.current.error).toBe(apiError);
  });

  it('useGitHubStatus: API rejection surfaces as hook error', async () => {
    const fakeApi = {
      getGitHubStatus: vi.fn().mockRejectedValue(apiError),
      connectGitHub: vi.fn(),
      disconnectGitHub: vi.fn(),
      listGitHubForks: vi.fn(),
      createGitHubFork: vi.fn(),
      getPRDiff: vi.fn(),
      createPullRequest: vi.fn(),
      saveGitHubPAT: vi.fn(),
      getGitHubPATStatus: vi.fn(),
      deleteGitHubPAT: vi.fn(),
    };
    const adapter = createGitHubAdapter(fakeApi);

    const { result } = renderHook(() => useGitHubStatus(adapter), { wrapper: createWrapper() });

    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(result.current.error).toBe(apiError);
  });

  it('useKeys: API rejection surfaces as hook error', async () => {
    const fakeApi = {
      listKeys: vi.fn().mockRejectedValue(apiError),
      createKey: vi.fn(),
      deleteKey: vi.fn(),
    };
    const adapter = createKeysAdapter(fakeApi);

    const { result } = renderHook(() => useKeys('proj', adapter), { wrapper: createWrapper() });

    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(result.current.error).toBe(apiError);
  });

  it('useWorkspaceList: API rejection surfaces as hook error', async () => {
    const fakeApi = {
      listWorkspace: vi.fn().mockRejectedValue(apiError),
      readWorkspaceFile: vi.fn(),
      writeWorkspaceFile: vi.fn(),
      getSessionGitHubDiff: vi.fn(),
      pushSessionToGitHub: vi.fn(),
      abandonSessionChanges: vi.fn(),
      getGitMergeStatus: vi.fn(),
      gitCreateBranch: vi.fn(),
      gitListBranches: vi.fn(),
      gitStatus: vi.fn(),
      configureGitRemote: vi.fn(),
    };
    const adapter = createSessionWorkspaceAdapter(fakeApi);

    const { result } = renderHook(
      () => useWorkspaceList('proj', 'sess', '/', undefined, adapter),
      { wrapper: createWrapper() },
    );

    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(result.current.error).toBe(apiError);
  });

  it('useScheduledSessions: API rejection surfaces as hook error', async () => {
    const fakeApi = {
      listScheduledSessions: vi.fn().mockRejectedValue(apiError),
      getScheduledSession: vi.fn(),
      createScheduledSession: vi.fn(),
      updateScheduledSession: vi.fn(),
      deleteScheduledSession: vi.fn(),
      suspendScheduledSession: vi.fn(),
      resumeScheduledSession: vi.fn(),
      triggerScheduledSession: vi.fn(),
      listScheduledSessionRuns: vi.fn(),
    };
    const adapter = createScheduledSessionsAdapter(fakeApi);

    const { result } = renderHook(
      () => useScheduledSessions('proj', adapter),
      { wrapper: createWrapper() },
    );

    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(result.current.error).toBe(apiError);
  });

  it('useSecretsList: API rejection surfaces as hook error', async () => {
    const fakeApi = {
      getSecretsList: vi.fn().mockRejectedValue(apiError),
      getSecretsConfig: vi.fn(),
      getSecretsValues: vi.fn(),
      updateSecretsConfig: vi.fn(),
      updateSecrets: vi.fn(),
      getIntegrationSecrets: vi.fn(),
      updateIntegrationSecrets: vi.fn(),
    };
    const adapter = createSecretsAdapter(fakeApi);

    const { result } = renderHook(() => useSecretsList('proj', adapter), { wrapper: createWrapper() });

    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(result.current.error).toBe(apiError);
  });

  it('useProjectAccess: API rejection surfaces as hook error', async () => {
    const fakeApi = {
      getProjectAccess: vi.fn().mockRejectedValue(apiError),
      getProjectPermissions: vi.fn(),
      addProjectPermission: vi.fn(),
      removeProjectPermission: vi.fn(),
    };
    const adapter = createProjectAccessAdapter(fakeApi);

    const { result } = renderHook(
      () => useProjectAccess('proj', adapter),
      { wrapper: createWrapper() },
    );

    await waitFor(() => expect(result.current.isError).toBe(true), { timeout: 10000 });
    expect(result.current.error).toBe(apiError);
  });
});

describe('error paths: mutation adapters propagate errors', () => {
  it('useCreateSession: API rejection surfaces as mutation error', async () => {
    const fakeApi = {
      listSessionsPaginated: vi.fn(),
      getSession: vi.fn(),
      createSession: vi.fn().mockRejectedValue(apiError),
      stopSession: vi.fn(),
      startSession: vi.fn(),
      cloneSession: vi.fn(),
      deleteSession: vi.fn(),
      getSessionPodEvents: vi.fn(),
      updateSessionDisplayName: vi.fn(),
      getSessionExport: vi.fn(),
      switchSessionModel: vi.fn(),
      saveToGoogleDrive: vi.fn(),
      listSessions: vi.fn(),
      getReposStatus: vi.fn(),
      getMcpStatus: vi.fn(),
      updateSessionMcpServers: vi.fn(),
      getCapabilities: vi.fn(),
    };
    const adapter = createSessionsAdapter(fakeApi);

    const { result } = renderHook(() => useCreateSession(adapter), { wrapper: createWrapper() });

    act(() => {
      result.current.mutate({
        projectName: 'proj',
        data: { prompt: 'hello' } as Parameters<typeof result.current.mutate>[0]['data'],
      });
    });

    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(result.current.error).toBe(apiError);
  });

  it('useStopSession: API rejection surfaces as mutation error', async () => {
    const fakeApi = {
      listSessionsPaginated: vi.fn(),
      getSession: vi.fn(),
      createSession: vi.fn(),
      stopSession: vi.fn().mockRejectedValue(apiError),
      startSession: vi.fn(),
      cloneSession: vi.fn(),
      deleteSession: vi.fn(),
      getSessionPodEvents: vi.fn(),
      updateSessionDisplayName: vi.fn(),
      getSessionExport: vi.fn(),
      switchSessionModel: vi.fn(),
      saveToGoogleDrive: vi.fn(),
      listSessions: vi.fn(),
      getReposStatus: vi.fn(),
      getMcpStatus: vi.fn(),
      updateSessionMcpServers: vi.fn(),
      getCapabilities: vi.fn(),
    };
    const adapter = createSessionsAdapter(fakeApi);

    const { result } = renderHook(() => useStopSession(adapter), { wrapper: createWrapper() });

    act(() => {
      result.current.mutate({ projectName: 'proj', sessionName: 'sess-1' });
    });

    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(result.current.error).toBe(apiError);
  });

  it('useDeleteProject: API rejection surfaces as mutation error', async () => {
    const fakeApi = {
      listProjectsPaginated: vi.fn(),
      getProject: vi.fn(),
      createProject: vi.fn(),
      updateProject: vi.fn(),
      deleteProject: vi.fn().mockRejectedValue(apiError),
      getProjectIntegrationStatus: vi.fn(),
      getProjectMcpServers: vi.fn(),
      updateProjectMcpServers: vi.fn(),
      listProjects: vi.fn(),
      getProjectAccess: vi.fn(),
      getProjectPermissions: vi.fn(),
      addProjectPermission: vi.fn(),
      removeProjectPermission: vi.fn(),
    };
    const adapter = createProjectsAdapter(fakeApi);

    const { result } = renderHook(() => useDeleteProject(adapter), { wrapper: createWrapper() });

    act(() => { result.current.mutate('my-project'); });

    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(result.current.error).toBe(apiError);
  });

  it('useConnectGitHub: API rejection surfaces as mutation error', async () => {
    const fakeApi = {
      getGitHubStatus: vi.fn(),
      connectGitHub: vi.fn().mockRejectedValue(apiError),
      disconnectGitHub: vi.fn(),
      listGitHubForks: vi.fn(),
      createGitHubFork: vi.fn(),
      getPRDiff: vi.fn(),
      createPullRequest: vi.fn(),
      saveGitHubPAT: vi.fn(),
      getGitHubPATStatus: vi.fn(),
      deleteGitHubPAT: vi.fn(),
    };
    const adapter = createGitHubAdapter(fakeApi);

    const { result } = renderHook(() => useConnectGitHub(adapter), { wrapper: createWrapper() });

    act(() => { result.current.mutate({ installationId: 123 }); });

    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(result.current.error).toBe(apiError);
  });

  it('useCreateKey: API rejection surfaces as mutation error', async () => {
    const fakeApi = {
      listKeys: vi.fn(),
      createKey: vi.fn().mockRejectedValue(apiError),
      deleteKey: vi.fn(),
    };
    const adapter = createKeysAdapter(fakeApi);

    const { result } = renderHook(() => useCreateKey(adapter), { wrapper: createWrapper() });

    act(() => {
      result.current.mutate({ projectName: 'proj', data: { name: 'new-key' } });
    });

    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(result.current.error).toBe(apiError);
  });

  it('useWriteWorkspaceFile: API rejection surfaces as mutation error', async () => {
    const fakeApi = {
      listWorkspace: vi.fn(),
      readWorkspaceFile: vi.fn(),
      writeWorkspaceFile: vi.fn().mockRejectedValue(apiError),
      getSessionGitHubDiff: vi.fn(),
      pushSessionToGitHub: vi.fn(),
      abandonSessionChanges: vi.fn(),
      getGitMergeStatus: vi.fn(),
      gitCreateBranch: vi.fn(),
      gitListBranches: vi.fn(),
      gitStatus: vi.fn(),
      configureGitRemote: vi.fn(),
    };
    const adapter = createSessionWorkspaceAdapter(fakeApi);

    const { result } = renderHook(() => useWriteWorkspaceFile(adapter), { wrapper: createWrapper() });

    act(() => {
      result.current.mutate({
        projectName: 'proj',
        sessionName: 'sess',
        path: 'file.txt',
        content: 'data',
      });
    });

    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(result.current.error).toBe(apiError);
  });

  it('useDeleteScheduledSession: API rejection surfaces as mutation error', async () => {
    const fakeApi = {
      listScheduledSessions: vi.fn(),
      getScheduledSession: vi.fn(),
      createScheduledSession: vi.fn(),
      updateScheduledSession: vi.fn(),
      deleteScheduledSession: vi.fn().mockRejectedValue(apiError),
      suspendScheduledSession: vi.fn(),
      resumeScheduledSession: vi.fn(),
      triggerScheduledSession: vi.fn(),
      listScheduledSessionRuns: vi.fn(),
    };
    const adapter = createScheduledSessionsAdapter(fakeApi);

    const { result } = renderHook(() => useDeleteScheduledSession(adapter), { wrapper: createWrapper() });

    act(() => {
      result.current.mutate({ projectName: 'proj', name: 'sched-1' });
    });

    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(result.current.error).toBe(apiError);
  });

  it('useUpdateSecrets: API rejection surfaces as mutation error', async () => {
    const fakeApi = {
      getSecretsList: vi.fn(),
      getSecretsConfig: vi.fn(),
      getSecretsValues: vi.fn(),
      updateSecretsConfig: vi.fn(),
      updateSecrets: vi.fn().mockRejectedValue(apiError),
      getIntegrationSecrets: vi.fn(),
      updateIntegrationSecrets: vi.fn(),
    };
    const adapter = createSecretsAdapter(fakeApi);

    const { result } = renderHook(() => useUpdateSecrets(adapter), { wrapper: createWrapper() });

    act(() => {
      result.current.mutate({
        projectName: 'proj',
        secrets: [{ key: 'SECRET', value: 'val' }],
      });
    });

    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(result.current.error).toBe(apiError);
  });
});

describe('error paths: method-mapping adapters propagate errors', () => {
  it('sessionTasksAdapter: stopTask error propagates through mapped method', async () => {
    const fakeApi = {
      stopBackgroundTask: vi.fn().mockRejectedValue(apiError),
      getTaskOutput: vi.fn(),
    };
    const adapter = createSessionTasksAdapter(fakeApi);

    await expect(adapter.stopTask('proj', 'sess', 'task-1')).rejects.toBe(apiError);
    expect(fakeApi.stopBackgroundTask).toHaveBeenCalledWith('proj', 'sess', 'task-1');
  });

  it('projectAccessAdapter: getAccess error propagates through mapped method', async () => {
    const fakeApi = {
      getProjectAccess: vi.fn().mockRejectedValue(apiError),
      getProjectPermissions: vi.fn(),
      addProjectPermission: vi.fn(),
      removeProjectPermission: vi.fn(),
    };
    const adapter = createProjectAccessAdapter(fakeApi);

    await expect(adapter.getAccess('proj')).rejects.toBe(apiError);
    expect(fakeApi.getProjectAccess).toHaveBeenCalledWith('proj');
  });
});

describe('error paths: error identity is preserved (not wrapped or transformed)', () => {
  it('custom error objects pass through unchanged', async () => {
    const customError = { code: 'FORBIDDEN', message: 'Access denied', status: 403 };
    const fakeApi = { getCurrentUser: vi.fn().mockRejectedValue(customError) };
    const adapter = createAuthAdapter(fakeApi);

    const { result } = renderHook(() => useCurrentUser(adapter), { wrapper: createWrapper() });

    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(result.current.error).toBe(customError);
  });

  it('string errors pass through unchanged', async () => {
    const fakeApi = {
      listKeys: vi.fn().mockRejectedValue('network timeout'),
      createKey: vi.fn(),
      deleteKey: vi.fn(),
    };
    const adapter = createKeysAdapter(fakeApi);

    const { result } = renderHook(() => useKeys('proj', adapter), { wrapper: createWrapper() });

    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(result.current.error).toBe('network timeout');
  });
});
