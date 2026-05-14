import * as authApi from '../../api/auth'
import type { AuthPort } from '../../ports/auth'

type AuthApi = typeof authApi

export function createAuthAdapter(api: AuthApi): AuthPort {
  return {
    getCurrentUser: api.getCurrentUser,
  }
}

export const authAdapter = createAuthAdapter(authApi)
