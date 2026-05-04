import * as jiraApi from '../../api/jira-auth'
import type { JiraPort } from '../../ports/jira'

type JiraApi = typeof jiraApi

export function createJiraAdapter(api: JiraApi): JiraPort {
  return {
    getJiraStatus: api.getJiraStatus,
    connectJira: api.connectJira,
    disconnectJira: api.disconnectJira,
  }
}

export const jiraAdapter = createJiraAdapter(jiraApi)
