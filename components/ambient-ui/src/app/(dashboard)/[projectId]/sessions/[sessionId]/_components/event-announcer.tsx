'use client'

import { useState, useEffect, useRef } from 'react'

type EventAnnouncerProps = {
  totalCount: number
  errorCount: number
}

/**
 * Visually hidden aria-live region that announces batched new event counts.
 * Debounced to at most every 10 seconds to avoid overwhelming screen readers.
 */
export function EventAnnouncer({ totalCount, errorCount }: EventAnnouncerProps) {
  const [announcement, setAnnouncement] = useState('')
  const prevTotalRef = useRef(totalCount)
  const prevErrorRef = useRef(errorCount)
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const pendingRef = useRef<{ newEvents: number; newErrors: number }>({
    newEvents: 0,
    newErrors: 0,
  })

  useEffect(() => {
    const newEvents = totalCount - prevTotalRef.current
    const newErrors = errorCount - prevErrorRef.current

    prevTotalRef.current = totalCount
    prevErrorRef.current = errorCount

    if (newEvents <= 0) return

    pendingRef.current.newEvents += newEvents
    pendingRef.current.newErrors += Math.max(0, newErrors)

    if (timerRef.current) return

    timerRef.current = setTimeout(() => {
      const { newEvents: pending, newErrors: pendingErrors } = pendingRef.current
      if (pending > 0) {
        const parts = [`${pending} new ${pending === 1 ? 'event' : 'events'}`]
        if (pendingErrors > 0) {
          parts.push(`${pendingErrors} ${pendingErrors === 1 ? 'error' : 'errors'}`)
        }
        setAnnouncement(parts.join(', '))
      }
      pendingRef.current = { newEvents: 0, newErrors: 0 }
      timerRef.current = null
    }, 10_000)
  }, [totalCount, errorCount])

  useEffect(() => {
    return () => {
      if (timerRef.current) {
        clearTimeout(timerRef.current)
      }
    }
  }, [])

  return (
    <div
      aria-live="polite"
      aria-atomic="true"
      className="sr-only"
    >
      {announcement}
    </div>
  )
}
