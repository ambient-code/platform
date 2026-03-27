"use client"

import { useMemo } from "react"
import { useQuery } from "@tanstack/react-query"
import { RefreshCw, Loader2 } from "lucide-react"
import { Button } from "@/components/ui/button"
import { StreamMessage } from "@/components/ui/stream-message"
import { getTaskOutput, stopBackgroundTask } from "@/services/api/tasks"
import { transformTaskTranscript } from "@/lib/transform-task-transcript"
import { StatusIcon, statusLabel, formatDuration, formatTokens } from "@/lib/task-utils"
import type { BackgroundTask } from "@/types/background-task"

type TaskTranscriptViewerProps = {
  projectName: string
  sessionName: string
  taskId: string
  task?: BackgroundTask
}

export function TaskTranscriptViewer({
  projectName,
  sessionName,
  taskId,
  task,
}: TaskTranscriptViewerProps) {
  const isRunning = task?.status === "running"

  const { data, isLoading, error, refetch, isFetching } = useQuery({
    queryKey: ["task-output", projectName, sessionName, taskId],
    queryFn: () => getTaskOutput(projectName, sessionName, taskId),
    refetchInterval: isRunning ? 5000 : false,
  })

  const messages = useMemo(
    () => (data ? transformTaskTranscript(data.output) : []),
    [data],
  )

  const handleStop = async () => {
    try {
      await stopBackgroundTask(projectName, sessionName, taskId)
    } catch (err) {
      console.error("Failed to stop task:", err)
    }
  }

  const status = task?.status ?? "completed"

  return (
    <div className="flex flex-col h-full">
      {/* Header */}
      <div className="flex items-center justify-between px-4 py-2 border-b bg-muted/30 flex-shrink-0">
        <div className="flex items-center gap-2 min-w-0 flex-1">
          <StatusIcon status={status} />
          <span className="text-sm font-medium truncate">
            {task?.description ?? taskId}
          </span>
          <span className="text-xs text-muted-foreground flex-shrink-0">
            {statusLabel(status)}
          </span>
          {task?.usage && (
            <span className="text-xs text-muted-foreground flex-shrink-0">
              · {formatDuration(task.usage.duration_ms)} · {formatTokens(task.usage.total_tokens)} tokens
            </span>
          )}
        </div>
        <div className="flex items-center gap-1 flex-shrink-0">
          {isRunning && (
            <Button variant="outline" size="sm" className="h-7 text-xs" onClick={handleStop}>
              Stop
            </Button>
          )}
          <Button
            variant="ghost"
            size="icon"
            className="h-7 w-7"
            onClick={() => refetch()}
            disabled={isFetching}
            title="Refresh transcript"
          >
            <RefreshCw className={`w-3.5 h-3.5 ${isFetching ? "animate-spin" : ""}`} />
          </Button>
        </div>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-y-auto">
        {isLoading ? (
          <div className="flex items-center justify-center py-12">
            <Loader2 className="h-5 w-5 animate-spin" />
            <span className="ml-2 text-sm text-muted-foreground">Loading transcript...</span>
          </div>
        ) : error ? (
          <div className="flex flex-col items-center justify-center py-12 text-sm text-muted-foreground">
            <p>Failed to load transcript</p>
            <Button variant="outline" size="sm" className="mt-2" onClick={() => refetch()}>
              Retry
            </Button>
          </div>
        ) : messages.length === 0 ? (
          <div className="flex items-center justify-center py-12 text-sm text-muted-foreground">
            {isRunning ? "Transcript will appear as the task runs..." : "No transcript data available"}
          </div>
        ) : (
          <div className="px-6 py-4 space-y-4">
            {messages.map((msg, i) => (
              <StreamMessage key={i} message={msg} plainCard />
            ))}
          </div>
        )}
      </div>
    </div>
  )
}
