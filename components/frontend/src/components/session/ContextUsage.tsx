/**
 * Context Usage Indicator
 *
 * Displays context window utilization in the session UI.
 */

import type { AGUIClientState } from '@/types/agui'
import { getContextLimit } from '@/types/agui'

function formatTokens(n: number): string {
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`
  if (n >= 1_000) return `${Math.round(n / 1_000)}K`
  return n.toString()
}

type ContextUsageProps = {
  state: AGUIClientState
  /** Model name to determine context limit (e.g., "claude-opus-4-5") */
  model?: string
  className?: string
}

/** Context usage: token count + sparkline */
export function ContextUsage({ state, model, className = '' }: ContextUsageProps) {
  const { used, perTurn } = state.contextUsage
  const limit = getContextLimit(model)
  const recent = perTurn.slice(-24)
  const max = Math.max(...recent, 1)

  return (
    <div className={`flex items-center gap-3 ${className}`}>
      <span className="text-muted-foreground font-mono text-sm">
        {formatTokens(used)}/{formatTokens(limit)}
      </span>

      {recent.length > 0 && (
        <div className="flex items-end gap-[2px] h-5">
          {recent.map((tokens, i) => (
            <div
              key={i}
              className="w-[3px] bg-green-500 rounded-sm"
              style={{ height: `${Math.max(8, (tokens / max) * 100)}%` }}
            />
          ))}
        </div>
      )}
    </div>
  )
}
