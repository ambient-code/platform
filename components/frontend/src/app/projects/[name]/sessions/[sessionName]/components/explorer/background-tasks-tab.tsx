"use client"

import { useState } from "react"
import { Loader2, CheckCircle2, XCircle, Square } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { cn } from "@/lib/utils"
import { stopBackgroundTask, getTaskOutput } from "@/services/api/tasks"
import type { BackgroundTask, TaskOutputResponse } from "@/types/background-task"

type BackgroundTasksTabProps = {
  backgroundTasks: Map<string, BackgroundTask>
  projectName: string
  sessionName: string
}

function formatDuration(ms: number): string {
  if (ms < 1000) return `${ms}ms`
  const seconds = Math.floor(ms / 1000)
  if (seconds < 60) return `${seconds}s`
  const minutes = Math.floor(seconds / 60)
  return `${minutes}m ${seconds % 60}s`
}

function formatTokens(count: number): string {
  if (count < 1000) return String(count)
  return `${(count / 1000).toFixed(1)}k`
}

function StatusIcon({ status }: { status: BackgroundTask["status"] }) {
  switch (status) {
    case "running":
      return <Loader2 className="h-4 w-4 animate-spin text-blue-500 flex-shrink-0" />
    case "completed":
      return <CheckCircle2 className="h-4 w-4 text-green-500 flex-shrink-0" />
    case "failed":
      return <XCircle className="h-4 w-4 text-red-500 flex-shrink-0" />
    case "stopped":
      return <Square className="h-4 w-4 text-muted-foreground flex-shrink-0" />
  }
}

function statusLabel(status: BackgroundTask["status"]): string {
  return status.charAt(0).toUpperCase() + status.slice(1)
}

export function BackgroundTasksTab({
  backgroundTasks,
  projectName,
  sessionName,
}: BackgroundTasksTabProps) {
  const [stoppingTaskId, setStoppingTaskId] = useState<string | null>(null)
  const [transcriptOpen, setTranscriptOpen] = useState(false)
  const [transcriptData, setTranscriptData] = useState<TaskOutputResponse | null>(null)
  const [transcriptLoading, setTranscriptLoading] = useState(false)
  const [transcriptTaskDesc, setTranscriptTaskDesc] = useState("")

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

  const handleViewTranscript = async (task: BackgroundTask) => {
    setTranscriptTaskDesc(task.description)
    setTranscriptLoading(true)
    setTranscriptOpen(true)
    try {
      const output = await getTaskOutput(projectName, sessionName, task.task_id)
      setTranscriptData(output)
    } catch (err) {
      console.error("Failed to load transcript:", err)
      setTranscriptData(null)
    } finally {
      setTranscriptLoading(false)
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
    <>
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
                  {/* Usage stats */}
                  {task.usage && (
                    <div className="text-xs text-muted-foreground mt-0.5">
                      Tokens: {formatTokens(task.usage.total_tokens)}
                      {task.usage.tool_uses > 0 && (
                        <> · Tools: {task.usage.tool_uses}</>
                      )}
                    </div>
                  )}
                  {/* Summary for completed tasks */}
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
                {task.status !== "running" && (
                  <Button
                    variant="ghost"
                    size="sm"
                    className="h-6 text-xs px-2"
                    onClick={() => handleViewTranscript(task)}
                  >
                    View transcript
                  </Button>
                )}
              </div>
            </div>
          ))}
        </div>
      </div>

      <Dialog open={transcriptOpen} onOpenChange={setTranscriptOpen}>
        <DialogContent className="max-w-2xl max-h-[80vh] flex flex-col">
          <DialogHeader>
            <DialogTitle className="truncate">
              Transcript: {transcriptTaskDesc}
            </DialogTitle>
          </DialogHeader>
          <div className="flex-1 overflow-y-auto">
            {transcriptLoading ? (
              <div className="flex items-center justify-center py-8">
                <Loader2 className="h-5 w-5 animate-spin" />
                <span className="ml-2 text-sm text-muted-foreground">Loading transcript...</span>
              </div>
            ) : transcriptData && transcriptData.output.length > 0 ? (
              <div className="space-y-2">
                {transcriptData.output.map((entry, i) => (
                  <div key={i} className="rounded border p-2 text-xs">
                    {typeof entry.type === "string" && (
                      <Badge variant="secondary" className="mb-1 text-[10px]">
                        {entry.type}
                      </Badge>
                    )}
                    <pre className="whitespace-pre-wrap break-words font-mono text-[11px]">
                      {JSON.stringify(entry, null, 2)}
                    </pre>
                  </div>
                ))}
              </div>
            ) : (
              <div className="text-sm text-muted-foreground text-center py-8">
                No transcript data available
              </div>
            )}
          </div>
        </DialogContent>
      </Dialog>
    </>
  )
}
