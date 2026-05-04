import * as ldapApi from '../../api/ldap'
import type { LdapPort } from '../../ports/ldap'

type LdapApi = typeof ldapApi

export function createLdapAdapter(api: LdapApi): LdapPort {
  return {
    searchUsers: api.searchUsers,
    searchGroups: api.searchGroups,
    getUser: api.getUser,
  }
}

export const ldapAdapter = createLdapAdapter(ldapApi)
