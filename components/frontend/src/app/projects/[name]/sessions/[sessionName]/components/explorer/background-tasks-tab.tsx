"use client"

import { useState } from "react"
import { Loader2 } from "lucide-react"
import { Button } from "@/components/ui/button"
import { cn } from "@/lib/utils"
import { stopBackgroundTask } from "@/services/api/tasks"
import { StatusIcon, statusLabel, formatDuration, formatTokens } from "@/lib/task-utils"
import type { BackgroundTask } from "@/types/background-task"

type BackgroundTasksTabProps = {
  backgroundTasks: Map<string, BackgroundTask>
  projectName: string
  sessionName: string
  onOpenTranscript: (task: BackgroundTask) => void
}

export function BackgroundTasksTab({
  backgroundTasks,
  projectName,
  sessionName,
  onOpenTranscript,
}: BackgroundTasksTabProps) {
  const [stoppingTaskId, setStoppingTaskId] = useState<string | null>(null)

  const tasks = Array.from(backgroundTasks.values())

  const handleStop = async (taskId: string) => {
    setStoppingTaskId(taskId)
    try {
      await stopBackgroundTask(projectName, sessionName, taskId)
    } catch (err) {
      console.error("Failed to stop task:", err)
    } finally {
      setStoppingTaskId(null)
    }
  }

  if (tasks.length === 0) {
    return (
      <div className="flex items-center justify-center h-full text-sm text-muted-foreground p-4">
        No background tasks
      </div>
    )
  }

  const runningCount = tasks.filter((t) => t.status === "running").length

  return (
    <div className="flex flex-col h-full overflow-hidden">
      <div className="px-3 py-2 border-b text-xs font-medium text-muted-foreground">
        Background Tasks{runningCount > 0 ? ` (${runningCount} running)` : ""}
      </div>
      <div className="flex-1 overflow-y-auto">
        {tasks.map((task) => (
          <div
            key={task.task_id}
            className="border-b px-3 py-2.5 text-sm space-y-1"
          >
            <div className="flex items-start gap-2">
              <StatusIcon status={task.status} />
              <div className="flex-1 min-w-0">
                <div className="font-medium truncate">{task.description}</div>
                <div className="text-xs text-muted-foreground flex items-center gap-1.5 mt-0.5">
                  <span>{statusLabel(task.status)}</span>
                  {task.usage?.duration_ms != null && (
                    <>
                      <span>·</span>
                      <span>{formatDuration(task.usage.duration_ms)}</span>
                    </>
                  )}
                  {task.last_tool_name && task.status === "running" && (
                    <>
                      <span>·</span>
                      <span>{task.last_tool_name}</span>
                    </>
                  )}
                </div>
                {task.usage && (
                  <div className="text-xs text-muted-foreground mt-0.5">
                    Tokens: {formatTokens(task.usage.total_tokens)}
                    {task.usage.tool_uses > 0 && (
                      <> · Tools: {task.usage.tool_uses}</>
                    )}
                  </div>
                )}
                {task.summary && task.status !== "running" && (
                  <div className={cn(
                    "text-xs mt-1 italic",
                    task.status === "failed" ? "text-red-500" : "text-muted-foreground",
                  )}>
                    {task.summary}
                  </div>
                )}
              </div>
            </div>

            <div className="flex justify-end gap-1">
              {task.status === "running" && (
                <Button
                  variant="outline"
                  size="sm"
                  className="h-6 text-xs px-2"
                  disabled={stoppingTaskId === task.task_id}
                  onClick={() => handleStop(task.task_id)}
                >
                  {stoppingTaskId === task.task_id ? (
                    <Loader2 className="h-3 w-3 animate-spin mr-1" />
                  ) : null}
                  Stop
                </Button>
              )}
              <Button
                variant="ghost"
                size="sm"
                className="h-6 text-xs px-2"
                onClick={() => onOpenTranscript(task)}
              >
                View transcript
              </Button>
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}
