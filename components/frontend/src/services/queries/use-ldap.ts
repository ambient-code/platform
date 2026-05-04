import { useQuery } from '@tanstack/react-query';
import { ldapAdapter } from '../adapters/ldap';
import type { LdapPort } from '../ports/ldap';
import { BACKEND_VERSION } from './query-keys';

export const ldapKeys = {
  all: [BACKEND_VERSION, 'ldap'] as const,
  users: () => [...ldapKeys.all, 'users'] as const,
  userSearch: (query: string) => [...ldapKeys.users(), 'search', query] as const,
  user: (uid: string) => [...ldapKeys.users(), uid] as const,
  groups: () => [...ldapKeys.all, 'groups'] as const,
  groupSearch: (query: string) => [...ldapKeys.groups(), 'search', query] as const,
};

export function useLDAPUserSearch(query: string, port: LdapPort = ldapAdapter) {
  return useQuery({
    queryKey: ldapKeys.userSearch(query),
    queryFn: () => port.searchUsers(query),
    enabled: query.length >= 2,
    staleTime: 5 * 60 * 1000,
  });
}

export function useLDAPGroupSearch(query: string, port: LdapPort = ldapAdapter) {
  return useQuery({
    queryKey: ldapKeys.groupSearch(query),
    queryFn: () => port.searchGroups(query),
    enabled: query.length >= 2,
    staleTime: 5 * 60 * 1000,
  });
}

export function useLDAPUser(uid: string, port: LdapPort = ldapAdapter) {
  return useQuery({
    queryKey: ldapKeys.user(uid),
    queryFn: () => port.getUser(uid),
    enabled: !!uid,
    staleTime: 5 * 60 * 1000,
  });
}
