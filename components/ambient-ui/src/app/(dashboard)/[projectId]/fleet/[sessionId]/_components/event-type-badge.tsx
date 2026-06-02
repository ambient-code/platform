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
    className: 'bg-[#e0f0ff] text-[#003366] border-[#b9dafc]',
  },
  assistant: {
    label: 'Assistant',
    className: 'bg-[#ece6ff] text-[#21134d] border-[#d0c5f4]',
  },
  text: {
    label: 'Text',
    className: 'bg-[#e0e0e0] text-[#383838] border-[#c7c7c7]',
  },
  tool_use: {
    label: 'Tool Call',
    className: 'bg-[#e0f0ff] text-[#003366] border-[#b9dafc]',
  },
  tool_result: {
    label: 'Tool Result',
    className: 'bg-[#daf2f2] text-[#004d4d] border-[#b9e5e5]',
  },
  error: {
    label: 'Error',
    className: 'bg-[#ffe3d9] text-[#731f00] border-[#fbbea8]',
  },
  lifecycle: {
    label: 'Lifecycle',
    className: 'bg-[#ece6ff] text-[#21134d] border-[#d0c5f4]',
  },
  user_feedback: {
    label: 'Feedback',
    className: 'bg-[#e9f7df] text-[#204d00] border-[#d1f1bb]',
  },
  system: {
    label: 'System',
    className: 'bg-[#f2f2f2] text-[#4d4d4d] border-[#e0e0e0]',
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
