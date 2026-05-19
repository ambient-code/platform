import * as sessionsApi from '../../api/sessions'
import type { SessionReposPort } from '../../ports/session-repos'

type SessionReposApi = Pick<typeof sessionsApi, 'getReposStatus'>

export function createSessionReposAdapter(api: SessionReposApi): SessionReposPort {
  return {
    getReposStatus: api.getReposStatus,
  }
}

export const sessionReposAdapter = createSessionReposAdapter(sessionsApi)
