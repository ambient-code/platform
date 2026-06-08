'use client'

import { useRef, useState, useEffect, useCallback } from 'react'
import { Button } from '@/components/ui/button'

type LiveTailState = {
  scrollRef: React.RefObject<HTMLDivElement | null>
  sentinelRef: React.RefObject<HTMLDivElement | null>
  isAtBottom: boolean
  newEventCount: number
  scrollToBottom: () => void
}

export function useLiveTail(messageCount: number): LiveTailState {
  const scrollRef = useRef<HTMLDivElement | null>(null)
  const sentinelRef = useRef<HTMLDivElement | null>(null)
  const [isAtBottom, setIsAtBottom] = useState(true)
  const [newEventCount, setNewEventCount] = useState(0)
  const prevCountRef = useRef(messageCount)

  // Track whether the user is at the bottom via IntersectionObserver
  useEffect(() => {
    const sentinel = sentinelRef.current
    const container = scrollRef.current
    if (!sentinel || !container) return

    const observer = new IntersectionObserver(
      ([entry]) => {
        setIsAtBottom(entry.isIntersecting)
        if (entry.isIntersecting) {
          setNewEventCount(0)
        }
      },
      { root: container, threshold: 0.1 },
    )

    observer.observe(sentinel)
    return () => observer.disconnect()
  }, [])

  // Track new events arriving while scrolled up
  useEffect(() => {
    const diff = messageCount - prevCountRef.current
    if (diff > 0 && !isAtBottom) {
      setNewEventCount(prev => prev + diff)
    }
    prevCountRef.current = messageCount
  }, [messageCount, isAtBottom])

  // Auto-scroll when at bottom and new messages arrive
  useEffect(() => {
    if (isAtBottom && scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight
    }
  }, [messageCount, isAtBottom])

  const scrollToBottom = useCallback(() => {
    if (scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight
      setNewEventCount(0)
    }
  }, [])

  return { scrollRef, sentinelRef, isAtBottom, newEventCount, scrollToBottom }
}

export function LiveIndicator() {
  return (
    <span className="inline-flex items-center gap-1.5 text-xs font-medium text-emerald-600">
      <span
        className="h-2 w-2 rounded-full bg-emerald-500 animate-pulse"
        aria-hidden="true"
      />
      Live
    </span>
  )
}

type JumpToLatestPillProps = {
  newEventCount: number
  onClick: () => void
}

export function JumpToLatestPill({ newEventCount, onClick }: JumpToLatestPillProps) {
  if (newEventCount <= 0) return null

  return (
    <div className="sticky bottom-2 flex justify-center pointer-events-none">
      <Button
        variant="secondary"
        size="sm"
        className="pointer-events-auto shadow-md text-xs h-7 rounded-full px-4"
        onClick={onClick}
      >
        {newEventCount} new {newEventCount === 1 ? 'event' : 'events'} — Jump to latest
      </Button>
    </div>
  )
}
