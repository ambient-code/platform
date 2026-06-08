import type { DomainSession, SessionPhase } from '@/domain/types'

// Annotation keys
const REVIEW_STATUS_KEY = 'ambient-code.io/review/status'
const NEEDS_INPUT_KEY = 'ambient-code.io/agent/needs-input'
const JIRA_ISSUE_KEY = 'ambient-code.io/jira/issue'
const GITHUB_PR_KEY = 'ambient-code.io/github/pr'
const COST_ANNOTATION_KEY = 'ambient-code.io/cost/estimate'

const ACTIVE_PHASES: ReadonlySet<SessionPhase> = new Set([
  'Running',
  'Creating',
  'Pending',
  'Stopping',
])

const TERMINAL_PHASES: ReadonlySet<SessionPhase> = new Set([
  'Completed',
  'Failed',
  'Stopped',
])

export type AttentionReason = 'failed' | 'needs-review' | 'needs-input'

export type AttentionItem = {
  session: DomainSession
  reason: AttentionReason
}

export type WorkItemRef = {
  type: 'jira' | 'github-pr'
  key: string
}

export type WorkItemGroup = {
  ref: WorkItemRef
  sessions: DomainSession[]
}

export type RecentActivityItem = {
  session: DomainSession
  ref: WorkItemRef | null
  cost: string | null
}

/** Sessions that need operator attention */
export function getAttentionItems(sessions: DomainSession[]): AttentionItem[] {
  const items: AttentionItem[] = []

  for (const session of sessions) {
    if (session.phase === 'Failed') {
      items.push({ session, reason: 'failed' })
      continue
    }

    const reviewStatus = session.annotations[REVIEW_STATUS_KEY]
    if (reviewStatus === 'needs-review') {
      items.push({ session, reason: 'needs-review' })
      continue
    }

    const needsInput = session.annotations[NEEDS_INPUT_KEY]
    if (needsInput && needsInput !== 'false') {
      items.push({ session, reason: 'needs-input' })
    }
  }

  return items
}

/** Extract the primary work item reference from a session's annotations */
function getWorkItemRef(session: DomainSession): WorkItemRef | null {
  const jiraKey = session.annotations[JIRA_ISSUE_KEY]
  if (jiraKey) {
    return { type: 'jira', key: jiraKey }
  }

  const prRef = session.annotations[GITHUB_PR_KEY]
  if (prRef) {
    return { type: 'github-pr', key: prRef }
  }

  return null
}

/** Active sessions grouped by work item reference */
export function getActiveWorkItems(sessions: DomainSession[]): {
  grouped: WorkItemGroup[]
  ungrouped: DomainSession[]
} {
  const activeSessions = sessions.filter(s => ACTIVE_PHASES.has(s.phase))

  const groupMap = new Map<string, WorkItemGroup>()
  const ungrouped: DomainSession[] = []

  for (const session of activeSessions) {
    const ref = getWorkItemRef(session)
    if (!ref) {
      ungrouped.push(session)
      continue
    }

    const groupKey = `${ref.type}:${ref.key}`
    const existing = groupMap.get(groupKey)
    if (existing) {
      existing.sessions.push(session)
    } else {
      groupMap.set(groupKey, { ref, sessions: [session] })
    }
  }

  return {
    grouped: Array.from(groupMap.values()),
    ungrouped,
  }
}

const RECENT_ACTIVITY_LIMIT = 10

/** Recently completed sessions for the activity feed */
export function getRecentActivity(sessions: DomainSession[]): RecentActivityItem[] {
  const completed = sessions
    .filter(s => TERMINAL_PHASES.has(s.phase))
    .sort((a, b) => {
      const aTime = a.completionTime ?? a.updatedAt
      const bTime = b.completionTime ?? b.updatedAt
      return new Date(bTime).getTime() - new Date(aTime).getTime()
    })
    .slice(0, RECENT_ACTIVITY_LIMIT)

  return completed.map(session => ({
    session,
    ref: getWorkItemRef(session),
    cost: session.annotations[COST_ANNOTATION_KEY] ?? null,
  }))
}
