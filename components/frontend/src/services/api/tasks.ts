import { apiClient } from './client'
import type { TaskOutputResponse } from '@/types/background-task'

export async function stopBackgroundTask(
  project: string,
  session: string,
  taskId: string,
): Promise<{ message: string }> {
  return apiClient.post<{ message: string }>(
    `/projects/${project}/agentic-sessions/${session}/agui/tasks/${taskId}/stop`,
  )
}

export async function getTaskOutput(
  project: string,
  session: string,
  taskId: string,
): Promise<TaskOutputResponse> {
  return apiClient.get<TaskOutputResponse>(
    `/projects/${project}/agentic-sessions/${session}/agui/tasks/${taskId}/output`,
  )
}

