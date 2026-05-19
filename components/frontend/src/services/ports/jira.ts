import type { JiraStatus, JiraConnectRequest } from './types'

export type JiraPort = {
  getJiraStatus: () => Promise<JiraStatus>
  connectJira: (data: JiraConnectRequest) => Promise<void>
  disconnectJira: () => Promise<void>
}
