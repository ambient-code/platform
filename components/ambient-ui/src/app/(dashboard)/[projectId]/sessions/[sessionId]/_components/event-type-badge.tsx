import { Badge } from '@/components/ui/badge'
import type { SessionEventType } from '@/domain/types'
import { cn } from '@/lib/utils'

type EventBadgeConfig = {
  label: string
  className: string
}

export const EVENT_BADGE_CONFIG: Record<SessionEventType, EventBadgeConfig> = {
  user: {
    label: 'User',
    className: 'bg-event-user text-event-user-foreground border-event-user-border',
  },
  assistant: {
    label: 'Assistant',
    className: 'bg-event-assistant text-event-assistant-foreground border-event-assistant-border',
  },
  text: {
    label: 'Text',
    className: 'bg-event-lifecycle text-event-lifecycle-foreground border-event-lifecycle-border',
  },
  tool_use: {
    label: 'Tool Call',
    className: 'bg-event-tool text-event-tool-foreground border-event-tool-border',
  },
  tool_result: {
    label: 'Tool Result',
    className: 'bg-event-user text-event-user-foreground border-event-user-border',
  },
  error: {
    label: 'Error',
    className: 'bg-status-error text-status-error-foreground border-status-error-border',
  },
  lifecycle: {
    label: 'Lifecycle',
    className: 'bg-event-assistant text-event-assistant-foreground border-event-assistant-border',
  },
  user_feedback: {
    label: 'Feedback',
    className: 'bg-event-feedback text-event-feedback-foreground border-event-feedback-border',
  },
  system: {
    label: 'System',
    className: 'bg-event-system text-event-system-foreground border-event-system-border',
  },
}

const VALID_EVENT_TYPES = new Set<string>(Object.keys(EVENT_BADGE_CONFIG))

function resolveEventType(raw: string): SessionEventType {
  if (VALID_EVENT_TYPES.has(raw)) {
    return raw as SessionEventType
  }
  return 'system'
}

export function EventTypeBadge({ eventType }: { eventType: string }) {
  const resolved = resolveEventType(eventType)
  const config = EVENT_BADGE_CONFIG[resolved]

  return (
    <Badge
      variant="outline"
      className={cn('text-[11px] font-medium uppercase tracking-wider', config.className)}
    >
      {config.label}
    </Badge>
  )
}
