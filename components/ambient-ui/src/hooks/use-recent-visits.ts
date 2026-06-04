'use client'

import { useCallback, useSyncExternalStore } from 'react'

const STORAGE_KEY = 'ambient:recent-visits'
const MAX_ITEMS = 50

export type RecentVisitType = 'session' | 'agent' | 'credential' | 'project'

export type RecentVisitItem = {
  type: RecentVisitType
  id: string
  projectId: string | null
  label: string
  sublabel: string | null
  href: string
  visitedAt: string
}

// ---------------------------------------------------------------------------
// External-store plumbing so every consumer re-renders on writes
// ---------------------------------------------------------------------------
let listeners: Array<() => void> = []

function emitChange() {
  for (const listener of listeners) {
    listener()
  }
}

function subscribe(listener: () => void): () => void {
  listeners = [...listeners, listener]
  return () => {
    listeners = listeners.filter((l) => l !== listener)
  }
}

// ---------------------------------------------------------------------------
// Read / write helpers
// ---------------------------------------------------------------------------

const RECENT_VISIT_TYPES: ReadonlySet<RecentVisitType> = new Set([
  'session', 'agent', 'credential', 'project',
])

function readItems(): RecentVisitItem[] {
  if (typeof window === 'undefined') return []
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return []
    const parsed: unknown = JSON.parse(raw)
    if (!Array.isArray(parsed)) return []
    return (parsed as Record<string, unknown>[]).filter(
      (item): item is RecentVisitItem =>
        typeof item.id === 'string' &&
        typeof item.type === 'string' &&
        RECENT_VISIT_TYPES.has(item.type as RecentVisitType) &&
        typeof item.href === 'string' &&
        typeof item.label === 'string',
    ) as RecentVisitItem[]
  } catch {
    return []
  }
}

function writeItems(items: RecentVisitItem[]): void {
  if (typeof window === 'undefined') return
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(items))
  } catch {
    // storage full or blocked -- silently ignore
  }
}

// Snapshot reference for useSyncExternalStore -- only changes on writes
let snapshot: RecentVisitItem[] = readItems()

function getSnapshot(): RecentVisitItem[] {
  return snapshot
}

function getServerSnapshot(): RecentVisitItem[] {
  return []
}

// ---------------------------------------------------------------------------
// Hook
// ---------------------------------------------------------------------------

export function useRecentVisits() {
  const recentItems = useSyncExternalStore(subscribe, getSnapshot, getServerSnapshot)

  const recordVisit = useCallback(
    (item: Omit<RecentVisitItem, 'visitedAt'>) => {
      const now = new Date().toISOString()
      const current = readItems()

      // Remove any existing entry with the same type+id+projectId to avoid duplicates
      const filtered = current.filter(
        (existing) =>
          !(
            existing.type === item.type &&
            existing.id === item.id &&
            existing.projectId === item.projectId
          ),
      )

      const entry: RecentVisitItem = { ...item, visitedAt: now }

      // Prepend new entry, prune to max
      const updated = [entry, ...filtered].slice(0, MAX_ITEMS)

      writeItems(updated)
      snapshot = updated
      emitChange()
    },
    [],
  )

  return { recentItems, recordVisit } as const
}
