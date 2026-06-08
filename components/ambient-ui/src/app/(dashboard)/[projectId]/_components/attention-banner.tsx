import Link from 'next/link'
import { AlertTriangle, MessageCircle, HelpCircle } from 'lucide-react'
import { Badge } from '@/components/ui/badge'
import type { AttentionItem, AttentionReason } from './dashboard-helpers'

type AttentionBannerProps = {
  items: AttentionItem[]
  projectId: string
}

const REASON_CONFIG: Record<
  AttentionReason,
  { label: string; icon: typeof AlertTriangle; variant: 'destructive' | 'default' | 'secondary' }
> = {
  failed: {
    label: 'Failed',
    icon: AlertTriangle,
    variant: 'destructive',
  },
  'needs-review': {
    label: 'Needs review',
    icon: MessageCircle,
    variant: 'default',
  },
  'needs-input': {
    label: 'Needs input',
    icon: HelpCircle,
    variant: 'secondary',
  },
}

export function AttentionBanner({ items, projectId }: AttentionBannerProps) {
  if (items.length === 0) {
    return null
  }

  return (
    <div className="rounded-lg border border-status-warning-border bg-status-warning/30 p-4">
      <h2 className="mb-3 text-sm font-semibold">
        Needs attention ({items.length})
      </h2>
      <ul className="space-y-2">
        {items.map(item => {
          const config = REASON_CONFIG[item.reason]
          const Icon = config.icon

          return (
            <li key={item.session.id}>
              <Link
                href={`/${projectId}/sessions/${item.session.id}`}
                className="flex min-w-0 items-center gap-2 rounded-md px-2 py-1.5 text-sm transition-colors hover:bg-accent"
              >
                <Icon className="size-4 shrink-0 text-muted-foreground" />
                <span className="min-w-0 truncate font-medium text-link hover:text-link-hover">
                  {item.session.name}
                </span>
                <Badge variant={config.variant} className="ml-auto shrink-0 text-xs">
                  {config.label}
                </Badge>
              </Link>
            </li>
          )
        })}
      </ul>
    </div>
  )
}
