import { renderHook, waitFor, act } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import {
  secretsKeys,
  useSecretsList,
  useSecretsConfig,
  useSecretsValues,
  useUpdateSecretsConfig,
  useUpdateSecrets,
  useIntegrationSecrets,
  useUpdateIntegrationSecrets,
} from '../use-secrets';
import type { SecretsPort } from '../../ports/secrets';
import { createWrapper, createTestQueryClient } from './test-utils';
import { BACKEND_VERSION } from '../query-keys';

const mockSecretList = { secrets: ['SECRET_A', 'SECRET_B'] };
const mockSecretsConfig = { name: 'my-secret', namespace: 'proj-ns' };
const mockSecretValues = [
  { key: 'SECRET_A', value: 'val-a' },
  { key: 'SECRET_B', value: 'val-b' },
];
const mockIntegrationSecrets = [{ key: 'INTEGRATION_KEY', value: 'int-val' }];

function createFakeSecretsPort(overrides?: Partial<SecretsPort>): SecretsPort {
  return {
    getSecretsList: vi.fn().mockResolvedValue(mockSecretList),
    getSecretsConfig: vi.fn().mockResolvedValue(mockSecretsConfig),
    getSecretsValues: vi.fn().mockResolvedValue(mockSecretValues),
    updateSecretsConfig: vi.fn().mockResolvedValue(undefined),
    updateSecrets: vi.fn().mockResolvedValue(undefined),
    getIntegrationSecrets: vi.fn().mockResolvedValue(mockIntegrationSecrets),
    updateIntegrationSecrets: vi.fn().mockResolvedValue(undefined),
    ...overrides,
  };
}

describe('secretsKeys', () => {
  it('includes BACKEND_VERSION prefix in all keys', () => {
    expect(secretsKeys.all[0]).toBe(BACKEND_VERSION);
    expect(secretsKeys.all).toEqual([BACKEND_VERSION, 'secrets']);
  });

  it('generates correct list key', () => {
    expect(secretsKeys.lists()).toEqual([BACKEND_VERSION, 'secrets', 'list']);
    expect(secretsKeys.list('proj')).toEqual([BACKEND_VERSION, 'secrets', 'list', 'proj']);
  });

  it('generates correct config key', () => {
    expect(secretsKeys.configs()).toEqual([BACKEND_VERSION, 'secrets', 'config']);
    expect(secretsKeys.config('proj')).toEqual([BACKEND_VERSION, 'secrets', 'config', 'proj']);
  });

  it('generates correct values key', () => {
    expect(secretsKeys.values()).toEqual([BACKEND_VERSION, 'secrets', 'values']);
    expect(secretsKeys.valuesForProject('proj')).toEqual([BACKEND_VERSION, 'secrets', 'values', 'proj']);
  });

  it('generates correct integration key', () => {
    expect(secretsKeys.integrations()).toEqual([BACKEND_VERSION, 'integration-secrets']);
    expect(secretsKeys.integration('proj')).toEqual([BACKEND_VERSION, 'integration-secrets', 'proj']);
  });
});

describe('useSecretsList', () => {
  it('calls port.getSecretsList and returns data', async () => {
    const fakePort = createFakeSecretsPort();
    const { result } = renderHook(() => useSecretsList('proj', fakePort), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.getSecretsList).toHaveBeenCalledWith('proj');
    expect(result.current.data).toEqual(mockSecretList);
  });

  it('is disabled when projectName is empty', () => {
    const fakePort = createFakeSecretsPort();
    const { result } = renderHook(() => useSecretsList('', fakePort), {
      wrapper: createWrapper(),
    });

    expect(result.current.fetchStatus).toBe('idle');
    expect(fakePort.getSecretsList).not.toHaveBeenCalled();
  });

  it('uses the correct query key', async () => {
    const queryClient = createTestQueryClient();
    const fakePort = createFakeSecretsPort();
    const { result } = renderHook(() => useSecretsList('proj', fakePort), {
      wrapper: createWrapper(queryClient),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    const cache = queryClient.getQueryCache().findAll();
    expect(cache[0].queryKey).toEqual(secretsKeys.list('proj'));
  });
});

describe('useSecretsConfig', () => {
  it('calls port.getSecretsConfig and returns data', async () => {
    const fakePort = createFakeSecretsPort();
    const { result } = renderHook(() => useSecretsConfig('proj', fakePort), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.getSecretsConfig).toHaveBeenCalledWith('proj');
    expect(result.current.data).toEqual(mockSecretsConfig);
  });

  it('is disabled when projectName is empty', () => {
    const fakePort = createFakeSecretsPort();
    const { result } = renderHook(() => useSecretsConfig('', fakePort), {
      wrapper: createWrapper(),
    });

    expect(result.current.fetchStatus).toBe('idle');
    expect(fakePort.getSecretsConfig).not.toHaveBeenCalled();
  });
});

describe('useSecretsValues', () => {
  it('calls port.getSecretsValues and returns data', async () => {
    const fakePort = createFakeSecretsPort();
    const { result } = renderHook(() => useSecretsValues('proj', fakePort), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.getSecretsValues).toHaveBeenCalledWith('proj');
    expect(result.current.data).toEqual(mockSecretValues);
  });

  it('is disabled when projectName is empty', () => {
    const fakePort = createFakeSecretsPort();
    const { result } = renderHook(() => useSecretsValues('', fakePort), {
      wrapper: createWrapper(),
    });

    expect(result.current.fetchStatus).toBe('idle');
    expect(fakePort.getSecretsValues).not.toHaveBeenCalled();
  });
});

describe('useUpdateSecretsConfig', () => {
  it('calls port.updateSecretsConfig with correct args', async () => {
    const fakePort = createFakeSecretsPort();
    const { result } = renderHook(() => useUpdateSecretsConfig(fakePort), {
      wrapper: createWrapper(),
    });

    act(() => {
      result.current.mutate({ projectName: 'proj', secretName: 'my-secret' });
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.updateSecretsConfig).toHaveBeenCalledWith('proj', 'my-secret');
  });

  it('invalidates config and valuesForProject caches on success', async () => {
    const queryClient = createTestQueryClient();
    const fakePort = createFakeSecretsPort();

    const configKey = secretsKeys.config('proj');
    const valuesKey = secretsKeys.valuesForProject('proj');
    queryClient.setQueryData(configKey, mockSecretsConfig);
    queryClient.setQueryData(valuesKey, mockSecretValues);

    const { result } = renderHook(() => useUpdateSecretsConfig(fakePort), {
      wrapper: createWrapper(queryClient),
    });

    act(() => {
      result.current.mutate({ projectName: 'proj', secretName: 'my-secret' });
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(queryClient.getQueryState(configKey)?.isInvalidated).toBe(true);
    expect(queryClient.getQueryState(valuesKey)?.isInvalidated).toBe(true);
  });
});

describe('useUpdateSecrets', () => {
  it('calls port.updateSecrets with correct args', async () => {
    const fakePort = createFakeSecretsPort();
    const { result } = renderHook(() => useUpdateSecrets(fakePort), {
      wrapper: createWrapper(),
    });

    const secrets = [{ key: 'SECRET_A', value: 'new-val' }];
    act(() => {
      result.current.mutate({ projectName: 'proj', secrets });
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.updateSecrets).toHaveBeenCalledWith('proj', secrets);
  });

  it('invalidates valuesForProject cache on success', async () => {
    const queryClient = createTestQueryClient();
    const fakePort = createFakeSecretsPort();

    const valuesKey = secretsKeys.valuesForProject('proj');
    queryClient.setQueryData(valuesKey, mockSecretValues);

    const { result } = renderHook(() => useUpdateSecrets(fakePort), {
      wrapper: createWrapper(queryClient),
    });

    act(() => {
      result.current.mutate({ projectName: 'proj', secrets: [{ key: 'A', value: 'B' }] });
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(queryClient.getQueryState(valuesKey)?.isInvalidated).toBe(true);
  });
});

describe('useIntegrationSecrets', () => {
  it('calls port.getIntegrationSecrets and returns data', async () => {
    const fakePort = createFakeSecretsPort();
    const { result } = renderHook(() => useIntegrationSecrets('proj', fakePort), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.getIntegrationSecrets).toHaveBeenCalledWith('proj');
    expect(result.current.data).toEqual(mockIntegrationSecrets);
  });

  it('is disabled when projectName is empty', () => {
    const fakePort = createFakeSecretsPort();
    const { result } = renderHook(() => useIntegrationSecrets('', fakePort), {
      wrapper: createWrapper(),
    });

    expect(result.current.fetchStatus).toBe('idle');
    expect(fakePort.getIntegrationSecrets).not.toHaveBeenCalled();
  });

  it('uses the correct query key', async () => {
    const queryClient = createTestQueryClient();
    const fakePort = createFakeSecretsPort();
    const { result } = renderHook(() => useIntegrationSecrets('proj', fakePort), {
      wrapper: createWrapper(queryClient),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    const cache = queryClient.getQueryCache().findAll();
    expect(cache[0].queryKey).toEqual(secretsKeys.integration('proj'));
  });
});

describe('useUpdateIntegrationSecrets', () => {
  it('calls port.updateIntegrationSecrets with correct args', async () => {
    const fakePort = createFakeSecretsPort();
    const { result } = renderHook(() => useUpdateIntegrationSecrets(fakePort), {
      wrapper: createWrapper(),
    });

    const secrets = [{ key: 'INT_KEY', value: 'int-val' }];
    act(() => {
      result.current.mutate({ projectName: 'proj', secrets });
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.updateIntegrationSecrets).toHaveBeenCalledWith('proj', secrets);
  });

  it('invalidates integration cache on success', async () => {
    const queryClient = createTestQueryClient();
    const fakePort = createFakeSecretsPort();

    const integrationKey = secretsKeys.integration('proj');
    queryClient.setQueryData(integrationKey, mockIntegrationSecrets);

    const { result } = renderHook(() => useUpdateIntegrationSecrets(fakePort), {
      wrapper: createWrapper(queryClient),
    });

    act(() => {
      result.current.mutate({ projectName: 'proj', secrets: [{ key: 'X', value: 'Y' }] });
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(queryClient.getQueryState(integrationKey)?.isInvalidated).toBe(true);
  });
});
