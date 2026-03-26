import { Loader2, CheckCircle2, XCircle, StopCircle } from "lucide-react"
import type { BackgroundTaskStatus } from "@/types/background-task"

export function StatusIcon({ status, className }: { status: BackgroundTaskStatus; className?: string }) {
  const base = className ?? "h-4 w-4 flex-shrink-0"
  switch (status) {
    case "running":
      return <Loader2 className={`${base} animate-spin text-blue-500`} />
    case "completed":
      return <CheckCircle2 className={`${base} text-green-500`} />
    case "failed":
      return <XCircle className={`${base} text-red-500`} />
    case "stopped":
      return <StopCircle className={`${base} text-muted-foreground`} />
  }
}

export function statusLabel(status: BackgroundTaskStatus): string {
  return status.charAt(0).toUpperCase() + status.slice(1)
}

export function formatDuration(ms: number): string {
  if (ms < 1000) return `${ms}ms`
  const seconds = Math.floor(ms / 1000)
  if (seconds < 60) return `${seconds}s`
  const minutes = Math.floor(seconds / 60)
  return `${minutes}m ${seconds % 60}s`
}

export function formatTokens(count: number): string {
  if (count < 1000) return String(count)
  return `${(count / 1000).toFixed(1)}k`
}
