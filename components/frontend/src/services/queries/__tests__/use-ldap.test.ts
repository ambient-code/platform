import { renderHook, waitFor } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { ldapKeys, useLDAPUserSearch, useLDAPGroupSearch, useLDAPUser } from '../use-ldap';
import type { LdapPort } from '../../ports/ldap';
import { createWrapper, createTestQueryClient } from './test-utils';
import { BACKEND_VERSION } from '../query-keys';

const mockUsers = [
  { uid: 'jdoe', cn: 'John Doe', mail: 'jdoe@example.com' },
  { uid: 'asmith', cn: 'Alice Smith', mail: 'asmith@example.com' },
];
const mockGroups = [
  { cn: 'developers', dn: 'cn=developers,ou=groups,dc=example' },
  { cn: 'admins', dn: 'cn=admins,ou=groups,dc=example' },
];
const mockUser = { uid: 'jdoe', cn: 'John Doe', mail: 'jdoe@example.com' };

function createFakeLdapPort(overrides?: Partial<LdapPort>): LdapPort {
  return {
    searchUsers: vi.fn().mockResolvedValue(mockUsers),
    searchGroups: vi.fn().mockResolvedValue(mockGroups),
    getUser: vi.fn().mockResolvedValue(mockUser),
    ...overrides,
  };
}

describe('ldapKeys', () => {
  it('includes BACKEND_VERSION prefix in all keys', () => {
    expect(ldapKeys.all[0]).toBe(BACKEND_VERSION);
    expect(ldapKeys.all).toEqual([BACKEND_VERSION, 'ldap']);
  });

  it('generates correct users key', () => {
    expect(ldapKeys.users()).toEqual([BACKEND_VERSION, 'ldap', 'users']);
  });

  it('generates correct userSearch key', () => {
    expect(ldapKeys.userSearch('john')).toEqual([BACKEND_VERSION, 'ldap', 'users', 'search', 'john']);
  });

  it('generates correct user key', () => {
    expect(ldapKeys.user('jdoe')).toEqual([BACKEND_VERSION, 'ldap', 'users', 'jdoe']);
  });

  it('generates correct groups key', () => {
    expect(ldapKeys.groups()).toEqual([BACKEND_VERSION, 'ldap', 'groups']);
  });

  it('generates correct groupSearch key', () => {
    expect(ldapKeys.groupSearch('dev')).toEqual([BACKEND_VERSION, 'ldap', 'groups', 'search', 'dev']);
  });
});

describe('useLDAPUserSearch', () => {
  it('calls port.searchUsers and returns data', async () => {
    const fakePort = createFakeLdapPort();
    const { result } = renderHook(() => useLDAPUserSearch('john', fakePort), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.searchUsers).toHaveBeenCalledWith('john');
    expect(result.current.data).toEqual(mockUsers);
  });

  it('is disabled when query length is less than 2', () => {
    const fakePort = createFakeLdapPort();
    const { result } = renderHook(() => useLDAPUserSearch('j', fakePort), {
      wrapper: createWrapper(),
    });

    expect(result.current.fetchStatus).toBe('idle');
    expect(fakePort.searchUsers).not.toHaveBeenCalled();
  });

  it('is disabled when query is empty', () => {
    const fakePort = createFakeLdapPort();
    const { result } = renderHook(() => useLDAPUserSearch('', fakePort), {
      wrapper: createWrapper(),
    });

    expect(result.current.fetchStatus).toBe('idle');
    expect(fakePort.searchUsers).not.toHaveBeenCalled();
  });

  it('is enabled when query length is exactly 2', async () => {
    const fakePort = createFakeLdapPort();
    const { result } = renderHook(() => useLDAPUserSearch('jo', fakePort), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.searchUsers).toHaveBeenCalledWith('jo');
  });

  it('uses the correct query key', async () => {
    const queryClient = createTestQueryClient();
    const fakePort = createFakeLdapPort();
    const { result } = renderHook(() => useLDAPUserSearch('john', fakePort), {
      wrapper: createWrapper(queryClient),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    const cache = queryClient.getQueryCache().findAll();
    expect(cache[0].queryKey).toEqual(ldapKeys.userSearch('john'));
  });

  it('propagates errors from the port', async () => {
    const fakePort = createFakeLdapPort({
      searchUsers: vi.fn().mockRejectedValue(new Error('LDAP unavailable')),
    });
    const { result } = renderHook(() => useLDAPUserSearch('john', fakePort), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(result.current.error).toBeInstanceOf(Error);
  });
});

describe('useLDAPGroupSearch', () => {
  it('calls port.searchGroups and returns data', async () => {
    const fakePort = createFakeLdapPort();
    const { result } = renderHook(() => useLDAPGroupSearch('dev', fakePort), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.searchGroups).toHaveBeenCalledWith('dev');
    expect(result.current.data).toEqual(mockGroups);
  });

  it('is disabled when query length is less than 2', () => {
    const fakePort = createFakeLdapPort();
    const { result } = renderHook(() => useLDAPGroupSearch('d', fakePort), {
      wrapper: createWrapper(),
    });

    expect(result.current.fetchStatus).toBe('idle');
    expect(fakePort.searchGroups).not.toHaveBeenCalled();
  });

  it('is disabled when query is empty', () => {
    const fakePort = createFakeLdapPort();
    const { result } = renderHook(() => useLDAPGroupSearch('', fakePort), {
      wrapper: createWrapper(),
    });

    expect(result.current.fetchStatus).toBe('idle');
    expect(fakePort.searchGroups).not.toHaveBeenCalled();
  });

  it('is enabled when query length is exactly 2', async () => {
    const fakePort = createFakeLdapPort();
    const { result } = renderHook(() => useLDAPGroupSearch('de', fakePort), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.searchGroups).toHaveBeenCalledWith('de');
  });

  it('uses the correct query key', async () => {
    const queryClient = createTestQueryClient();
    const fakePort = createFakeLdapPort();
    const { result } = renderHook(() => useLDAPGroupSearch('dev', fakePort), {
      wrapper: createWrapper(queryClient),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    const cache = queryClient.getQueryCache().findAll();
    expect(cache[0].queryKey).toEqual(ldapKeys.groupSearch('dev'));
  });
});

describe('useLDAPUser', () => {
  it('calls port.getUser and returns data', async () => {
    const fakePort = createFakeLdapPort();
    const { result } = renderHook(() => useLDAPUser('jdoe', fakePort), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(fakePort.getUser).toHaveBeenCalledWith('jdoe');
    expect(result.current.data).toEqual(mockUser);
  });

  it('is disabled when uid is empty', () => {
    const fakePort = createFakeLdapPort();
    const { result } = renderHook(() => useLDAPUser('', fakePort), {
      wrapper: createWrapper(),
    });

    expect(result.current.fetchStatus).toBe('idle');
    expect(fakePort.getUser).not.toHaveBeenCalled();
  });

  it('uses the correct query key', async () => {
    const queryClient = createTestQueryClient();
    const fakePort = createFakeLdapPort();
    const { result } = renderHook(() => useLDAPUser('jdoe', fakePort), {
      wrapper: createWrapper(queryClient),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    const cache = queryClient.getQueryCache().findAll();
    expect(cache[0].queryKey).toEqual(ldapKeys.user('jdoe'));
  });

  it('propagates errors from the port', async () => {
    const fakePort = createFakeLdapPort({
      getUser: vi.fn().mockRejectedValue(new Error('User not found')),
    });
    const { result } = renderHook(() => useLDAPUser('unknown', fakePort), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(result.current.error).toBeInstanceOf(Error);
  });
});
