import { Badge } from '@/components/ui/badge'
import { cn } from '@/lib/utils'
import { getPhaseStyle } from '@/lib/status-colors'
import type { SessionPhase } from '@/domain/types'

const VARIANT_CLASSES = {
  success: 'bg-status-success text-status-success-foreground border-status-success-border',
  error: 'bg-status-error text-status-error-foreground border-status-error-border',
  warning: 'bg-status-warning text-status-warning-foreground border-status-warning-border',
  info: 'bg-status-info text-status-info-foreground border-status-info-border',
  default: 'bg-muted text-muted-foreground border-border',
} as const

export function PhaseDotOnly({ phase }: { phase: SessionPhase }) {
  const style = getPhaseStyle(phase)
  const dotColor: Record<string, string> = {
    success: 'bg-green-500',
    error: 'bg-red-500',
    warning: 'bg-amber-500',
    info: 'bg-blue-500',
    default: 'bg-muted-foreground',
  }
  return (
    <span
      className={cn('inline-block h-2.5 w-2.5 rounded-full shrink-0', dotColor[style.variant] ?? dotColor.default)}
      title={style.label}
    />
  )
}

export function PhaseBadge({ phase }: { phase: SessionPhase }) {
  const style = getPhaseStyle(phase)

  return (
    <Badge
      variant="outline"
      className={cn('gap-1.5 font-semibold', VARIANT_CLASSES[style.variant])}
    >
      {style.pulse && (
        <span className="relative flex h-2.5 w-2.5">
          <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-current opacity-75" />
          <span className="relative inline-flex h-2.5 w-2.5 rounded-full bg-current" />
        </span>
      )}
      {style.label}
    </Badge>
  )
}
