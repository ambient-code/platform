import type { LDAPUser, LDAPGroup } from './types'

export type LdapPort = {
  searchUsers: (query: string) => Promise<LDAPUser[]>
  searchGroups: (query: string) => Promise<LDAPGroup[]>
  getUser: (uid: string) => Promise<LDAPUser>
}
