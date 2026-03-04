/**
 * API service for LDAP user and group search
 */

import { apiClient } from './client';

// Types
export type LDAPUser = {
  uid: string;
  fullName: string;
  email: string;
  title: string;
  githubUsername: string;
  groups: string[];
};

export type LDAPGroup = {
  name: string;
  description: string;
};

type SearchUsersResponse = {
  users: LDAPUser[];
};

type SearchGroupsResponse = {
  groups: LDAPGroup[];
};

/**
 * Search LDAP users by query (min 2 chars)
 */
export async function searchUsers(query: string): Promise<LDAPUser[]> {
  const response = await apiClient.get<SearchUsersResponse>('/ldap/users', {
    params: { q: query },
  });
  return response.users || [];
}

/**
 * Search LDAP groups by query (min 2 chars)
 */
export async function searchGroups(query: string): Promise<LDAPGroup[]> {
  const response = await apiClient.get<SearchGroupsResponse>('/ldap/groups', {
    params: { q: query },
  });
  return response.groups || [];
}

/**
 * Get a single LDAP user by UID
 */
export async function getUser(uid: string): Promise<LDAPUser> {
  return apiClient.get<LDAPUser>(`/ldap/users/${encodeURIComponent(uid)}`);
}
