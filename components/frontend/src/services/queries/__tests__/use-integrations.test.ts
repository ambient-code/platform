import { renderHook, waitFor } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { useIntegrationsStatus, integrationsKeys } from '../use-integrations';
import type { IntegrationsPort } from '../../ports/integrations';
import { createWrapper, createTestQueryClient } from './test-utils';
import { BACKEND_VERSION } from '../query-keys';

const mockIntegrationsStatus = {
  github: {
    installed: true,
    pat: { configured: false },
  },
  google: { connected: false },
  jira: { connected: false },
  gitlab: { connected: false },
  coderabbit: { connected: false },
  gerrit: { connected: false },
};

function createFakeIntegrationsPort(
  overrides?: Partial<IntegrationsPort>,
): IntegrationsPort {
  return {
    getIntegrationsStatus: vi.fn().mockResolvedValue(mockIntegrationsStatus),
    ...overrides,
  };
}

describe('integrationsKeys', () => {
  it('includes BACKEND_VERSION prefix', () => {
    expect(integrationsKeys.all[0]).toBe(BACKEND_VERSION);
    expect(integrationsKeys.all).toEqual([BACKEND_VERSION, 'integrations']);
  });

  it('status key extends all', () => {
    expect(integrationsKeys.status()).toEqual([
      BACKEND_VERSION,
      'integrations',
      'status',
    ]);
  });
});

describe('useIntegrationsStatus', () => {
  it('calls port.getIntegrationsStatus and returns data', async () => {
    const fakePort = createFakeIntegrationsPort();
    const { result } = renderHook(() => useIntegrationsStatus(fakePort), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.getIntegrationsStatus).toHaveBeenCalledOnce();
    expect(result.current.data).toEqual(mockIntegrationsStatus);
  });

  it('uses the correct cache key', async () => {
    const queryClient = createTestQueryClient();
    const fakePort = createFakeIntegrationsPort();
    const { result } = renderHook(() => useIntegrationsStatus(fakePort), {
      wrapper: createWrapper(queryClient),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    const cache = queryClient.getQueryCache().findAll();
    expect(cache).toHaveLength(1);
    expect(cache[0].queryKey).toEqual(integrationsKeys.status());
  });

  it('propagates errors from the port', async () => {
    const fakePort = createFakeIntegrationsPort({
      getIntegrationsStatus: vi
        .fn()
        .mockRejectedValue(new Error('Failed to fetch integrations')),
    });
    const { result } = renderHook(() => useIntegrationsStatus(fakePort), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(result.current.error).toBeInstanceOf(Error);
  });
});
