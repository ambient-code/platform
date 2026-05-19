import * as sessionEventsApi from '../../api/session-events'
import type { SessionEventsPort } from '../../ports/session-events'

type SessionEventsApi = typeof sessionEventsApi

export function createSessionEventsAdapter(api: SessionEventsApi = sessionEventsApi): SessionEventsPort {
  return {
    createEventSource: api.createEventSource,
    sendMessage: api.sendMessage,
    interrupt: api.interrupt,
  }
}

export const sessionEventsAdapter = createSessionEventsAdapter()
