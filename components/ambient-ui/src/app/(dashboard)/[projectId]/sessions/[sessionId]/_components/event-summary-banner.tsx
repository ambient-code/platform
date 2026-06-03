import type { SessionEventType } from '@/domain/types'
import { cn } from '@/lib/utils'

type EventSummaryBannerProps = {
  totalCount: number
  filteredCount: number
  errorCount: number
  isFiltered: boolean
}

export function EventSummaryBanner({
  totalCount,
  filteredCount,
  errorCount,
  isFiltered,
}: EventSummaryBannerProps) {
  return (
    <div className="flex items-center justify-between text-sm text-muted-foreground">
      <span>
        {totalCount} {totalCount === 1 ? 'event' : 'events'}
        {errorCount > 0 && (
          <>
            {' — '}
            <span className="font-medium text-[#f0561d]">
              {errorCount} {errorCount === 1 ? 'error' : 'errors'}
            </span>
          </>
        )}
      </span>
      {isFiltered && (
        <span className="text-xs">
          Showing {filteredCount} of {totalCount} events
        </span>
      )}
    </div>
  )
}

export function computeEventCounts(
  messages: Array<{ eventType: string }>,
  eventTypes: readonly SessionEventType[],
): Record<SessionEventType, number> {
  const counts = {} as Record<SessionEventType, number>
  for (const et of eventTypes) {
    counts[et] = 0
  }
  for (const msg of messages) {
    const et = msg.eventType as SessionEventType
    if (et in counts) {
      counts[et]++
    }
  }
  return counts
}
