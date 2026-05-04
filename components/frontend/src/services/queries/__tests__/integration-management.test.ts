import { renderHook, waitFor, act } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { createKeysAdapter } from '../../adapters/keys';
import { createSecretsAdapter } from '../../adapters/secrets';
import { createModelsAdapter } from '../../adapters/models';
import { createRunnerTypesAdapter } from '../../adapters/runner-types';
import { createWorkflowsAdapter } from '../../adapters/workflows';
import { createFeatureFlagsAdapter } from '../../adapters/feature-flags';
import { createLdapAdapter } from '../../adapters/ldap';
import { createRepoAdapter } from '../../adapters/repo';
import { createClusterAdapter } from '../../adapters/cluster';
import { createVersionAdapter } from '../../adapters/version';
import { createConfigAdapter } from '../../adapters/config';
import { useKeys, useCreateKey, useDeleteKey } from '../use-keys';
import {
  useSecretsList,
  useSecretsConfig,
  useSecretsValues,
  useUpdateSecretsConfig,
  useUpdateSecrets,
  useIntegrationSecrets,
  useUpdateIntegrationSecrets,
} from '../use-secrets';
import { useModels } from '../use-models';
import { useRunnerTypes } from '../use-runner-types';
import { useOOTBWorkflows, useWorkflowMetadata } from '../use-workflows';
import {
  useFeatureFlags,
  useFeatureFlag,
  useWorkspaceFlag,
  useToggleFeatureFlag,
  useSetFeatureFlagOverride,
  useRemoveFeatureFlagOverride,
  useEnableFeatureFlag,
  useDisableFeatureFlag,
} from '../use-feature-flags-admin';
import { useLDAPUserSearch, useLDAPGroupSearch, useLDAPUser } from '../use-ldap';
import { useRepoBlob, useRepoTree, useRepoFileExists, useRepoBranches } from '../use-repo';
import { useClusterInfo } from '../use-cluster';
import { useVersion } from '../use-version';
import { useLoadingTips } from '../use-loading-tips';
import { createWrapper } from './test-utils';

describe('integration: hook → keysAdapter → fakeApi', () => {
  function createFakeKeysApi() {
    return {
      listKeys: vi.fn().mockResolvedValue([{ id: 'key-1', name: 'my-key', createdAt: '2026-01-01' }]),
      createKey: vi.fn().mockResolvedValue({ id: 'key-2', name: 'new-key' }),
      deleteKey: vi.fn().mockResolvedValue(undefined),
    };
  }

  it('useKeys: flows through', async () => {
    const fakeApi = createFakeKeysApi();
    const adapter = createKeysAdapter(fakeApi);

    const { result } = renderHook(() => useKeys('proj', adapter), { wrapper: createWrapper() });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.listKeys).toHaveBeenCalledWith('proj');
    expect(result.current.data).toHaveLength(1);
  });

  it('useCreateKey: flows through', async () => {
    const fakeApi = createFakeKeysApi();
    const adapter = createKeysAdapter(fakeApi);

    const { result } = renderHook(() => useCreateKey(adapter), { wrapper: createWrapper() });

    act(() => {
      result.current.mutate({ projectName: 'proj', data: { name: 'new-key' } });
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.createKey).toHaveBeenCalledWith('proj', { name: 'new-key' });
  });

  it('useDeleteKey: flows through', async () => {
    const fakeApi = createFakeKeysApi();
    const adapter = createKeysAdapter(fakeApi);

    const { result } = renderHook(() => useDeleteKey(adapter), { wrapper: createWrapper() });

    act(() => {
      result.current.mutate({ projectName: 'proj', keyId: 'key-1' });
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.deleteKey).toHaveBeenCalledWith('proj', 'key-1');
  });
});

describe('integration: hook → secretsAdapter → fakeApi', () => {
  function createFakeSecretsApi() {
    return {
      getSecretsList: vi.fn().mockResolvedValue(['SECRET_A', 'SECRET_B']),
      getSecretsConfig: vi.fn().mockResolvedValue({ secretName: 'my-secrets' }),
      getSecretsValues: vi.fn().mockResolvedValue([{ key: 'SECRET_A', value: '***' }]),
      updateSecretsConfig: vi.fn().mockResolvedValue(undefined),
      updateSecrets: vi.fn().mockResolvedValue(undefined),
      getIntegrationSecrets: vi.fn().mockResolvedValue([{ name: 'INT_KEY', value: '***' }]),
      updateIntegrationSecrets: vi.fn().mockResolvedValue(undefined),
    };
  }

  it('useSecretsList: flows through', async () => {
    const fakeApi = createFakeSecretsApi();
    const adapter = createSecretsAdapter(fakeApi);

    const { result } = renderHook(() => useSecretsList('proj', adapter), { wrapper: createWrapper() });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.getSecretsList).toHaveBeenCalledWith('proj');
    expect(result.current.data).toHaveLength(2);
  });

  it('useSecretsConfig: flows through', async () => {
    const fakeApi = createFakeSecretsApi();
    const adapter = createSecretsAdapter(fakeApi);

    const { result } = renderHook(() => useSecretsConfig('proj', adapter), { wrapper: createWrapper() });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.getSecretsConfig).toHaveBeenCalledWith('proj');
    expect(result.current.data).toEqual({ secretName: 'my-secrets' });
  });

  it('useSecretsValues: flows through', async () => {
    const fakeApi = createFakeSecretsApi();
    const adapter = createSecretsAdapter(fakeApi);

    const { result } = renderHook(() => useSecretsValues('proj', adapter), { wrapper: createWrapper() });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.getSecretsValues).toHaveBeenCalledWith('proj');
    expect(result.current.data).toHaveLength(1);
    expect(result.current.data?.[0].key).toBe('SECRET_A');
  });

  it('useUpdateSecretsConfig: flows through', async () => {
    const fakeApi = createFakeSecretsApi();
    const adapter = createSecretsAdapter(fakeApi);

    const { result } = renderHook(() => useUpdateSecretsConfig(adapter), { wrapper: createWrapper() });

    act(() => {
      result.current.mutate({ projectName: 'proj', secretName: 'new-secret' });
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.updateSecretsConfig).toHaveBeenCalledWith('proj', 'new-secret');
  });

  it('useUpdateSecrets: flows through', async () => {
    const fakeApi = createFakeSecretsApi();
    const adapter = createSecretsAdapter(fakeApi);

    const { result } = renderHook(() => useUpdateSecrets(adapter), { wrapper: createWrapper() });

    act(() => {
      result.current.mutate({
        projectName: 'proj',
        secrets: [{ key: 'SECRET_A', value: 'new-value' }],
      });
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.updateSecrets).toHaveBeenCalledWith('proj', [{ key: 'SECRET_A', value: 'new-value' }]);
  });

  it('useIntegrationSecrets: flows through', async () => {
    const fakeApi = createFakeSecretsApi();
    const adapter = createSecretsAdapter(fakeApi);

    const { result } = renderHook(() => useIntegrationSecrets('proj', adapter), { wrapper: createWrapper() });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.getIntegrationSecrets).toHaveBeenCalledWith('proj');
    expect(result.current.data).toHaveLength(1);
  });

  it('useUpdateIntegrationSecrets: flows through', async () => {
    const fakeApi = createFakeSecretsApi();
    const adapter = createSecretsAdapter(fakeApi);

    const { result } = renderHook(() => useUpdateIntegrationSecrets(adapter), { wrapper: createWrapper() });

    act(() => {
      result.current.mutate({
        projectName: 'proj',
        secrets: [{ key: 'INT_KEY', value: 'updated' }],
      });
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.updateIntegrationSecrets).toHaveBeenCalledWith('proj', [{ key: 'INT_KEY', value: 'updated' }]);
  });
});

describe('integration: hook → modelsAdapter → fakeApi', () => {
  it('useModels: flows through', async () => {
    const fakeApi = { getModelsForProject: vi.fn().mockResolvedValue([{ id: 'claude-sonnet-4-20250514', name: 'Sonnet' }]) };
    const adapter = createModelsAdapter(fakeApi);

    const { result } = renderHook(
      () => useModels('proj', true, undefined, adapter),
      { wrapper: createWrapper() },
    );

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.getModelsForProject).toHaveBeenCalledWith('proj', undefined);
    expect(result.current.data).toHaveLength(1);
  });
});

describe('integration: hook → runnerTypesAdapter → fakeApi', () => {
  it('useRunnerTypes: flows through', async () => {
    const fakeApi = { getRunnerTypes: vi.fn().mockResolvedValue([{ name: 'default', displayName: 'Default Runner' }]) };
    const adapter = createRunnerTypesAdapter(fakeApi);

    const { result } = renderHook(() => useRunnerTypes('proj', adapter), { wrapper: createWrapper() });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.getRunnerTypes).toHaveBeenCalledWith('proj');
    expect(result.current.data).toHaveLength(1);
  });
});

describe('integration: hook → workflowsAdapter → fakeApi', () => {
  function createFakeWorkflowsApi() {
    return {
      listOOTBWorkflows: vi.fn().mockResolvedValue([{ id: 'wf-1', name: 'Code Review' }]),
      getWorkflowMetadata: vi.fn().mockResolvedValue({ commands: [{ name: 'review' }], agents: [] }),
    };
  }

  it('useOOTBWorkflows: flows through', async () => {
    const fakeApi = createFakeWorkflowsApi();
    const adapter = createWorkflowsAdapter(fakeApi);

    const { result } = renderHook(() => useOOTBWorkflows('proj', adapter), { wrapper: createWrapper() });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.listOOTBWorkflows).toHaveBeenCalledWith('proj');
    expect(result.current.data).toHaveLength(1);
  });

  it('useWorkflowMetadata: flows through', async () => {
    const fakeApi = createFakeWorkflowsApi();
    const adapter = createWorkflowsAdapter(fakeApi);

    const { result } = renderHook(
      () => useWorkflowMetadata('proj', 'sess-1', true, adapter),
      { wrapper: createWrapper() },
    );

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.getWorkflowMetadata).toHaveBeenCalledWith('proj', 'sess-1');
    expect(result.current.data?.commands).toHaveLength(1);
  });
});

describe('integration: hook → featureFlagsAdapter → fakeApi', () => {
  function createFakeFeatureFlagsApi() {
    return {
      getFeatureFlags: vi.fn().mockResolvedValue([{ name: 'dark-mode', enabled: true }]),
      evaluateFeatureFlag: vi.fn().mockResolvedValue({ enabled: true, source: 'override' }),
      getFeatureFlag: vi.fn().mockResolvedValue({ name: 'dark-mode', enabled: true, description: 'Dark mode toggle' }),
      setFeatureFlagOverride: vi.fn().mockResolvedValue(undefined),
      removeFeatureFlagOverride: vi.fn().mockResolvedValue(undefined),
      enableFeatureFlag: vi.fn().mockResolvedValue(undefined),
      disableFeatureFlag: vi.fn().mockResolvedValue(undefined),
    };
  }

  it('useFeatureFlags: flows through', async () => {
    const fakeApi = createFakeFeatureFlagsApi();
    const adapter = createFeatureFlagsAdapter(fakeApi);

    const { result } = renderHook(() => useFeatureFlags('proj', adapter), { wrapper: createWrapper() });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.getFeatureFlags).toHaveBeenCalledWith('proj');
    expect(result.current.data).toHaveLength(1);
  });

  it('useFeatureFlag: flows through', async () => {
    const fakeApi = createFakeFeatureFlagsApi();
    const adapter = createFeatureFlagsAdapter(fakeApi);

    const { result } = renderHook(
      () => useFeatureFlag('proj', 'dark-mode', adapter),
      { wrapper: createWrapper() },
    );

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.getFeatureFlag).toHaveBeenCalledWith('proj', 'dark-mode');
    expect(result.current.data?.name).toBe('dark-mode');
  });

  it('useToggleFeatureFlag: enable flows through', async () => {
    const fakeApi = createFakeFeatureFlagsApi();
    const adapter = createFeatureFlagsAdapter(fakeApi);

    const { result } = renderHook(() => useToggleFeatureFlag(adapter), { wrapper: createWrapper() });

    act(() => {
      result.current.mutate({ projectName: 'proj', flagName: 'dark-mode', enable: true });
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.enableFeatureFlag).toHaveBeenCalledWith('proj', 'dark-mode');
  });

  it('useToggleFeatureFlag: disable flows through', async () => {
    const fakeApi = createFakeFeatureFlagsApi();
    const adapter = createFeatureFlagsAdapter(fakeApi);

    const { result } = renderHook(() => useToggleFeatureFlag(adapter), { wrapper: createWrapper() });

    act(() => {
      result.current.mutate({ projectName: 'proj', flagName: 'dark-mode', enable: false });
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.disableFeatureFlag).toHaveBeenCalledWith('proj', 'dark-mode');
  });

  it('useSetFeatureFlagOverride: flows through', async () => {
    const fakeApi = createFakeFeatureFlagsApi();
    const adapter = createFeatureFlagsAdapter(fakeApi);

    const { result } = renderHook(() => useSetFeatureFlagOverride(adapter), { wrapper: createWrapper() });

    act(() => {
      result.current.mutate({ projectName: 'proj', flagName: 'dark-mode', enabled: true });
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.setFeatureFlagOverride).toHaveBeenCalledWith('proj', 'dark-mode', true);
  });

  it('useRemoveFeatureFlagOverride: flows through', async () => {
    const fakeApi = createFakeFeatureFlagsApi();
    const adapter = createFeatureFlagsAdapter(fakeApi);

    const { result } = renderHook(() => useRemoveFeatureFlagOverride(adapter), { wrapper: createWrapper() });

    act(() => {
      result.current.mutate({ projectName: 'proj', flagName: 'dark-mode' });
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.removeFeatureFlagOverride).toHaveBeenCalledWith('proj', 'dark-mode');
  });

  it('useWorkspaceFlag: evaluate flows through with transformed result', async () => {
    const fakeApi = createFakeFeatureFlagsApi();
    const adapter = createFeatureFlagsAdapter(fakeApi);

    const { result } = renderHook(
      () => useWorkspaceFlag('proj', 'dark-mode', adapter),
      { wrapper: createWrapper() },
    );

    await waitFor(() => expect(result.current.enabled).toBe(true));
    expect(fakeApi.evaluateFeatureFlag).toHaveBeenCalledWith('proj', 'dark-mode');
    expect(result.current.source).toBe('override');
    expect(result.current.isLoading).toBe(false);
  });

  it('useEnableFeatureFlag: enable flows through', async () => {
    const fakeApi = createFakeFeatureFlagsApi();
    const adapter = createFeatureFlagsAdapter(fakeApi);

    const { result } = renderHook(() => useEnableFeatureFlag(adapter), { wrapper: createWrapper() });

    act(() => {
      result.current.mutate({ projectName: 'proj', flagName: 'dark-mode' });
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.enableFeatureFlag).toHaveBeenCalledWith('proj', 'dark-mode');
  });

  it('useDisableFeatureFlag: disable flows through', async () => {
    const fakeApi = createFakeFeatureFlagsApi();
    const adapter = createFeatureFlagsAdapter(fakeApi);

    const { result } = renderHook(() => useDisableFeatureFlag(adapter), { wrapper: createWrapper() });

    act(() => {
      result.current.mutate({ projectName: 'proj', flagName: 'dark-mode' });
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.disableFeatureFlag).toHaveBeenCalledWith('proj', 'dark-mode');
  });
});

describe('integration: hook → ldapAdapter → fakeApi', () => {
  function createFakeLdapApi() {
    return {
      searchUsers: vi.fn().mockResolvedValue([{ uid: 'user1', displayName: 'User One' }]),
      searchGroups: vi.fn().mockResolvedValue([{ cn: 'devs', displayName: 'Developers' }]),
      getUser: vi.fn().mockResolvedValue({ uid: 'user1', displayName: 'User One', email: 'user1@example.com' }),
    };
  }

  it('useLDAPUserSearch: flows through', async () => {
    const fakeApi = createFakeLdapApi();
    const adapter = createLdapAdapter(fakeApi);

    const { result } = renderHook(() => useLDAPUserSearch('user', adapter), { wrapper: createWrapper() });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.searchUsers).toHaveBeenCalledWith('user');
    expect(result.current.data).toHaveLength(1);
  });

  it('useLDAPGroupSearch: flows through', async () => {
    const fakeApi = createFakeLdapApi();
    const adapter = createLdapAdapter(fakeApi);

    const { result } = renderHook(() => useLDAPGroupSearch('dev', adapter), { wrapper: createWrapper() });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.searchGroups).toHaveBeenCalledWith('dev');
    expect(result.current.data).toHaveLength(1);
  });

  it('useLDAPUser: flows through', async () => {
    const fakeApi = createFakeLdapApi();
    const adapter = createLdapAdapter(fakeApi);

    const { result } = renderHook(() => useLDAPUser('user1', adapter), { wrapper: createWrapper() });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.getUser).toHaveBeenCalledWith('user1');
    expect(result.current.data?.uid).toBe('user1');
  });
});

describe('integration: hook → repoAdapter → fakeApi', () => {
  function createFakeRepoApi() {
    return {
      getRepoBlob: vi.fn().mockResolvedValue(new Response('file content')),
      getRepoTree: vi.fn().mockResolvedValue({ entries: [{ type: 'blob', name: 'README.md', path: 'README.md' }] }),
      checkFileExists: vi.fn().mockResolvedValue(true),
      listRepoBranches: vi.fn().mockResolvedValue({ branches: ['main', 'develop'] }),
    };
  }

  it('useRepoBlob: flows through', async () => {
    const fakeApi = createFakeRepoApi();
    const adapter = createRepoAdapter(fakeApi);
    const params = { repo: 'org/repo', ref: 'main', path: 'src/index.ts' };

    const { result } = renderHook(
      () => useRepoBlob('proj', params, undefined, adapter),
      { wrapper: createWrapper() },
    );

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.getRepoBlob).toHaveBeenCalledWith('proj', params);
    expect(result.current.data).toBeInstanceOf(Response);
  });

  it('useRepoTree: flows through', async () => {
    const fakeApi = createFakeRepoApi();
    const adapter = createRepoAdapter(fakeApi);
    const params = { repo: 'org/repo', ref: 'main', path: '/' };

    const { result } = renderHook(
      () => useRepoTree('proj', params, undefined, adapter),
      { wrapper: createWrapper() },
    );

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.getRepoTree).toHaveBeenCalledWith('proj', params);
    expect(result.current.data?.entries).toHaveLength(1);
  });

  it('useRepoFileExists: flows through', async () => {
    const fakeApi = createFakeRepoApi();
    const adapter = createRepoAdapter(fakeApi);
    const params = { repo: 'org/repo', ref: 'main', path: 'README.md' };

    const { result } = renderHook(
      () => useRepoFileExists('proj', params, undefined, adapter),
      { wrapper: createWrapper() },
    );

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.checkFileExists).toHaveBeenCalledWith('proj', params);
    expect(result.current.data).toBe(true);
  });

  it('useRepoBranches: flows through', async () => {
    const fakeApi = createFakeRepoApi();
    const adapter = createRepoAdapter(fakeApi);

    const { result } = renderHook(
      () => useRepoBranches('proj', 'org/repo', undefined, adapter),
      { wrapper: createWrapper() },
    );

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.listRepoBranches).toHaveBeenCalledWith('proj', 'org/repo');
    expect(result.current.data?.branches).toEqual(['main', 'develop']);
  });
});

describe('integration: hook → clusterAdapter → fakeApi', () => {
  it('useClusterInfo: flows through', async () => {
    const fakeApi = { getClusterInfo: vi.fn().mockResolvedValue({ isOpenShift: false, vertexEnabled: true }) };
    const adapter = createClusterAdapter(fakeApi);

    const { result } = renderHook(() => useClusterInfo(adapter), { wrapper: createWrapper() });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.getClusterInfo).toHaveBeenCalled();
    expect(result.current.data?.isOpenShift).toBe(false);
  });
});

describe('integration: hook → versionAdapter → fakeApi', () => {
  it('useVersion: flows through', async () => {
    const fakeApi = { getVersion: vi.fn().mockResolvedValue('2.1.0') };
    const adapter = createVersionAdapter(fakeApi);

    const { result } = renderHook(() => useVersion(adapter), { wrapper: createWrapper() });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.getVersion).toHaveBeenCalled();
    expect(result.current.data).toBe('2.1.0');
  });
});

describe('integration: hook → configAdapter → fakeApi', () => {
  it('useLoadingTips: flows through', async () => {
    const fakeApi = { getLoadingTips: vi.fn().mockResolvedValue(['Tip 1: Be patient', 'Tip 2: Stay calm']) };
    const adapter = createConfigAdapter(fakeApi);

    const { result } = renderHook(() => useLoadingTips(adapter), { wrapper: createWrapper() });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakeApi.getLoadingTips).toHaveBeenCalled();
    expect(result.current.data).toHaveLength(2);
  });
});
