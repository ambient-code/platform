'use client'

import { useState, useCallback } from 'react'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import type { DomainSession, DomainSessionMessage, SessionEventType } from '@/domain/types'
import { useSessionMessages } from '@/queries/use-session-messages'
import { formatRelativeTime } from '@/lib/format-timestamp'
import { cn } from '@/lib/utils'
import { EventTypeBadge } from './event-type-badge'

const OPERATIONAL_EVENT_TYPES: SessionEventType[] = [
  'tool_use',
  'tool_result',
  'error',
  'lifecycle',
  'system',
]

const EVENT_TYPE_LABELS: Record<string, string> = {
  tool_use: 'Tool Call',
  tool_result: 'Tool Result',
  error: 'Error',
  lifecycle: 'Lifecycle',
  system: 'System',
}

const MAX_CONTENT_LENGTH = 300

function truncateContent(content: string): { text: string; truncated: boolean } {
  if (content.length <= MAX_CONTENT_LENGTH) {
    return { text: content, truncated: false }
  }
  return {
    text: content.slice(0, MAX_CONTENT_LENGTH),
    truncated: true,
  }
}

function EventRow({ message }: { message: DomainSessionMessage }) {
  const [expanded, setExpanded] = useState(false)
  const isError = message.eventType === 'error'
  const { text, truncated } = truncateContent(message.payload)

  const toggleExpanded = useCallback(() => {
    setExpanded(prev => !prev)
  }, [])

  return (
    <div
      className={cn(
        'flex gap-3 px-3 py-2 text-sm',
        isError && 'border-l-2 border-l-[#f0561d] bg-[#ffe3d9]/20',
      )}
    >
      <span className="shrink-0 font-mono text-xs text-muted-foreground pt-0.5 min-w-[100px]">
        {message.createdAt ? formatRelativeTime(message.createdAt) : '--'}
      </span>
      <span className="shrink-0 pt-0.5">
        <EventTypeBadge eventType={message.eventType} />
      </span>
      <div className="min-w-0 flex-1">
        <pre className="whitespace-pre-wrap break-words font-mono text-xs text-foreground">
          {expanded ? message.payload : text}
          {truncated && !expanded && '...'}
        </pre>
        {truncated && (
          <Button
            variant="link"
            size="sm"
            className="h-auto p-0 text-xs text-muted-foreground"
            onClick={toggleExpanded}
          >
            {expanded ? 'Show less' : 'Show more'}
          </Button>
        )}
      </div>
    </div>
  )
}

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
    activeFilters.has(m.eventType as SessionEventType),
  )

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
      {/* Filter bar */}
      <div className="flex flex-wrap gap-2" role="group" aria-label="Filter by event type">
        {OPERATIONAL_EVENT_TYPES.map(eventType => {
          const isActive = activeFilters.has(eventType)
          return (
            <Button
              key={eventType}
              variant={isActive ? 'default' : 'outline'}
              size="sm"
              className="h-7 text-xs"
              onClick={() => toggleFilter(eventType)}
              aria-pressed={isActive}
            >
              {EVENT_TYPE_LABELS[eventType]}
            </Button>
          )
        })}
      </div>

      {/* Event list */}
      <Card>
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
            <div className="divide-y">
              {filteredMessages.map(message => (
                <EventRow key={message.id} message={message} />
              ))}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
