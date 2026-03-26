export type BackgroundTaskStatus = "running" | "completed" | "failed" | "stopped"

export type BackgroundTaskUsage = {
  total_tokens: number
  tool_uses: number
  duration_ms: number
}

export type BackgroundTask = {
  task_id: string
  description: string
  task_type?: string
  status: BackgroundTaskStatus
  usage?: BackgroundTaskUsage
  summary?: string
  last_tool_name?: string
  output_file?: string
}

export type TaskOutputEntry = Record<string, unknown>

export type TaskOutputResponse = {
  task_id: string
  output: TaskOutputEntry[]
}
