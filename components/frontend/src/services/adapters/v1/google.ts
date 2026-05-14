import * as googleApi from '../../api/google-auth'
import type { GooglePort } from '../../ports/google'

type GoogleApi = typeof googleApi

export function createGoogleAdapter(api: GoogleApi): GooglePort {
  return {
    getGoogleOAuthURL: api.getGoogleOAuthURL,
    getGoogleStatus: api.getGoogleStatus,
    disconnectGoogle: api.disconnectGoogle,
  }
}

export const googleAdapter = createGoogleAdapter(googleApi)
