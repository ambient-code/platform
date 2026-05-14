import * as tasksApi from '../../api/tasks'
import type { SessionTasksPort } from '../../ports/session-tasks'

type TasksApi = typeof tasksApi

export function createSessionTasksAdapter(api: TasksApi): SessionTasksPort {
  return {
    stopTask: api.stopBackgroundTask,
    getTaskOutput: api.getTaskOutput,
  }
}

export const sessionTasksAdapter = createSessionTasksAdapter(tasksApi)
