import type { TaskOutputResponse } from './types'

export type SessionTasksPort = {
  stopTask: (projectName: string, sessionName: string, taskId: string) => Promise<{ message: string }>
  getTaskOutput: (projectName: string, sessionName: string, taskId: string) => Promise<TaskOutputResponse>
}
