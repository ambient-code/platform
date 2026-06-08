import { formatDistanceToNow, formatDistance, format } from 'date-fns'

export function formatRelativeTime(iso: string): string {
  return formatDistanceToNow(new Date(iso), { addSuffix: true })
}

export function formatAbsoluteTime(iso: string): string {
  return format(new Date(iso), 'MMM d, yyyy h:mm:ss a')
}

export function formatDuration(startIso: string, endIso?: string | null): string {
  const start = new Date(startIso)
  const end = endIso ? new Date(endIso) : new Date()
  return formatDistance(start, end)
}

export function formatPreciseDuration(startIso: string, endIso?: string | null): string {
  const start = new Date(startIso)
  const end = endIso ? new Date(endIso) : new Date()
  const diffMs = Math.max(0, end.getTime() - start.getTime())
  const seconds = Math.floor(diffMs / 1000)
  const minutes = Math.floor(seconds / 60)
  const hours = Math.floor(minutes / 60)
  const days = Math.floor(hours / 24)

  if (days > 0) return `${days}d ${hours % 24}h`
  if (hours > 0) return `${hours}h ${minutes % 60}m`
  if (minutes > 0) return `${minutes}m ${seconds % 60}s`
  return `${seconds}s`
}
