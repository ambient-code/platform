import type { UserProfile } from './types'

export type AuthPort = {
  getCurrentUser: () => Promise<UserProfile>
}
