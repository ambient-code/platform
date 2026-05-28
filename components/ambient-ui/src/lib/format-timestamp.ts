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
