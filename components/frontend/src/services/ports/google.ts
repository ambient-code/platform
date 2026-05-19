import type { GoogleOAuthStatus, GoogleOAuthURLResponse } from './types'

export type GooglePort = {
  getGoogleOAuthURL: () => Promise<GoogleOAuthURLResponse>
  getGoogleStatus: () => Promise<GoogleOAuthStatus>
  disconnectGoogle: () => Promise<void>
}
