'use client'

import { useState, useCallback } from 'react'
import { AlertTriangle } from 'lucide-react'
import { Button } from '@/components/ui/button'
import type { DomainSessionMessage } from '@/domain/types'
import { formatRelativeTime } from '@/lib/format-timestamp'
import { cn } from '@/lib/utils'
import { EventTypeBadge } from './event-type-badge'

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

type EventRowProps = {
  message: DomainSessionMessage
  isToolResultFollowingToolUse: boolean
}

export function EventRow({ message, isToolResultFollowingToolUse }: EventRowProps) {
  const [expanded, setExpanded] = useState(false)
  const isError = message.eventType === 'error'
  const { text, truncated } = truncateContent(message.payload)

  const toggleExpanded = useCallback(() => {
    setExpanded(prev => !prev)
  }, [])

  const relativeTime = message.createdAt ? formatRelativeTime(message.createdAt) : '--'
  const ariaLabel = `${message.eventType === 'tool_use' ? 'Tool Call' : message.eventType === 'tool_result' ? 'Tool Result' : message.eventType} event, ${relativeTime}`

  return (
    <article
      aria-label={ariaLabel}
      className={cn(
        'flex gap-3 px-3 py-2 text-sm',
        isError && 'border-l-2 border-l-[#f0561d] bg-[#ffe3d9]/20',
        isToolResultFollowingToolUse && 'mt-0 pt-1 border-l-2 border-l-[#e0e0e0] ml-4',
      )}
    >
      <span className="shrink-0 font-mono text-xs text-muted-foreground pt-0.5 min-w-[100px]">
        {relativeTime}
      </span>
      <span className="shrink-0 pt-0.5 flex items-center gap-1">
        {isError && (
          <AlertTriangle className="h-3.5 w-3.5 text-[#f0561d]" aria-hidden="true" />
        )}
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
    </article>
  )
}
