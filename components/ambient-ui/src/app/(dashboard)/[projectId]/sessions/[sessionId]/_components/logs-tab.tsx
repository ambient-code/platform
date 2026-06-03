'use client'

import { useState, useCallback, useMemo } from 'react'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import type { DomainSession, SessionEventType } from '@/domain/types'
import { useSessionMessages } from '@/queries/use-session-messages'
import { cn } from '@/lib/utils'
import { EventRow } from './event-row'
import { EVENT_BADGE_CONFIG } from './event-type-badge'
import { EventSummaryBanner, computeEventCounts } from './event-summary-banner'
import { useLiveTail, LiveIndicator, JumpToLatestPill } from './live-tail-indicator'
import { EventAnnouncer } from './event-announcer'

const OPERATIONAL_EVENT_TYPES: readonly SessionEventType[] = [
  'tool_use',
  'tool_result',
  'error',
  'lifecycle',
  'system',
] as const

export function LogsTab({ session }: { session: DomainSession }) {
  const [activeFilters, setActiveFilters] = useState<Set<SessionEventType>>(
    new Set(OPERATIONAL_EVENT_TYPES),
  )

  const { data, isLoading, error } = useSessionMessages(session.id)

  const toggleFilter = useCallback((eventType: SessionEventType) => {
    setActiveFilters(prev => {
      const next = new Set(prev)
      if (next.has(eventType)) {
        next.delete(eventType)
      } else {
        next.add(eventType)
      }
      return next
    })
  }, [])

  const messages = data?.items ?? []
  const filteredMessages = messages.filter(m =>
    activeFilters.has(m.eventType),
  )

  const eventCounts = useMemo(
    () => computeEventCounts(messages, OPERATIONAL_EVENT_TYPES),
    [messages],
  )

  const isFiltered = activeFilters.size !== OPERATIONAL_EVENT_TYPES.length

  const { scrollRef, sentinelRef, isAtBottom, newEventCount, scrollToBottom } =
    useLiveTail(filteredMessages.length)

  if (error) {
    return (
      <div className="pt-4">
        <p className="text-sm text-destructive">
          Failed to load messages: {error.message}
        </p>
      </div>
    )
  }

  return (
    <div className="space-y-4 pt-4">
      <EventAnnouncer
        totalCount={messages.length}
        errorCount={eventCounts.error}
      />

      {!isLoading && messages.length > 0 && (
        <EventSummaryBanner
          totalCount={messages.length}
          filteredCount={filteredMessages.length}
          errorCount={eventCounts.error}
          isFiltered={isFiltered}
        />
      )}

      <div className="flex flex-wrap gap-2" role="group" aria-label="Filter by event type">
        {OPERATIONAL_EVENT_TYPES.map(eventType => {
          const isActive = activeFilters.has(eventType)
          const count = eventCounts[eventType]
          const isErrorType = eventType === 'error'
          const hasErrors = isErrorType && count > 0
          return (
            <Button
              key={eventType}
              variant={isActive ? 'default' : 'outline'}
              size="sm"
              className={cn(
                'h-7 text-xs gap-1.5',
                hasErrors && !isActive && 'border-[#f0561d] text-[#f0561d]',
              )}
              onClick={() => toggleFilter(eventType)}
              aria-pressed={isActive}
            >
              {EVENT_BADGE_CONFIG[eventType].label}
              {count > 0 && (
                <span
                  className={cn(
                    'inline-flex items-center justify-center rounded-full px-1.5 min-w-[1.25rem] h-4 text-[10px] font-medium',
                    isActive
                      ? 'bg-primary-foreground/20 text-primary-foreground'
                      : 'bg-muted text-muted-foreground',
                    hasErrors && !isActive && 'bg-[#ffe3d9] text-[#f0561d]',
                    hasErrors && isActive && 'bg-[#f0561d]/30 text-primary-foreground',
                  )}
                >
                  {count}
                </span>
              )}
            </Button>
          )
        })}
      </div>

      {/* Event list card */}
      <Card className="relative">
        {isAtBottom && filteredMessages.length > 0 && (
          <div className="absolute top-2 right-3 z-10">
            <LiveIndicator />
          </div>
        )}

        <CardContent className="p-0">
          {isLoading ? (
            <div className="space-y-2 p-4">
              <Skeleton className="h-8 w-full" />
              <Skeleton className="h-8 w-full" />
              <Skeleton className="h-8 w-full" />
            </div>
          ) : filteredMessages.length === 0 ? (
            <p className="p-6 text-center text-sm text-muted-foreground">
              {messages.length === 0
                ? 'No events recorded yet.'
                : 'No events match the selected filters.'}
            </p>
          ) : (
            <div
              ref={scrollRef}
              className="max-h-[600px] overflow-y-auto relative"
              role="log"
              aria-label="Session events"
            >
              <div className="divide-y">
                {filteredMessages.map((message, index) => {
                  const prev = index > 0 ? filteredMessages[index - 1] : null
                  const isToolResultFollowingToolUse =
                    message.eventType === 'tool_result' &&
                    prev?.eventType === 'tool_use'

                  return (
                    <EventRow
                      key={message.id}
                      message={message}
                      isToolResultFollowingToolUse={isToolResultFollowingToolUse}
                    />
                  )
                })}
              </div>
              {/* Sentinel for IntersectionObserver */}
              <div ref={sentinelRef} className="h-1" aria-hidden="true" />

              <JumpToLatestPill
                newEventCount={newEventCount}
                onClick={scrollToBottom}
              />
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
