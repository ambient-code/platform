/**
 * React Query hooks for LDAP user and group search
 */

import { useQuery } from '@tanstack/react-query';
import * as ldapApi from '../api/ldap';

// Query key factory
export const ldapKeys = {
  all: ['ldap'] as const,
  users: () => [...ldapKeys.all, 'users'] as const,
  userSearch: (query: string) => [...ldapKeys.users(), 'search', query] as const,
  user: (uid: string) => [...ldapKeys.users(), uid] as const,
  groups: () => [...ldapKeys.all, 'groups'] as const,
  groupSearch: (query: string) => [...ldapKeys.groups(), 'search', query] as const,
};

/**
 * Hook to search LDAP users with autocomplete
 * Only fires when query is >= 2 characters
 */
export function useLDAPUserSearch(query: string) {
  return useQuery({
    queryKey: ldapKeys.userSearch(query),
    queryFn: () => ldapApi.searchUsers(query),
    enabled: query.length >= 2,
    staleTime: 5 * 60 * 1000, // 5 minutes
  });
}

/**
 * Hook to search LDAP groups with autocomplete
 * Only fires when query is >= 2 characters
 */
export function useLDAPGroupSearch(query: string) {
  return useQuery({
    queryKey: ldapKeys.groupSearch(query),
    queryFn: () => ldapApi.searchGroups(query),
    enabled: query.length >= 2,
    staleTime: 5 * 60 * 1000, // 5 minutes
  });
}

/**
 * Hook to get a single LDAP user by UID
 */
export function useLDAPUser(uid: string) {
  return useQuery({
    queryKey: ldapKeys.user(uid),
    queryFn: () => ldapApi.getUser(uid),
    enabled: !!uid,
    staleTime: 5 * 60 * 1000,
  });
}
