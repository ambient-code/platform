import type { Session, Project } from 'ambient-sdk'
import type {
  DomainSession, DomainProject, DomainSessionMessage, SessionPhase, SessionEventType,
  DomainRepo, DomainReconciledRepo, DomainCondition, ReconciledRepoStatus, ConditionStatus,
} from '@/domain/types'

const VALID_PHASES: ReadonlySet<string> = new Set<string>([
  'Pending',
  'Creating',
  'Running',
  'Stopping',
  'Completed',
  'Failed',
  'Stopped',
])

function parsePhase(raw: string): SessionPhase {
  if (VALID_PHASES.has(raw)) {
    return raw as SessionPhase
  }
  return 'Pending'
}

function parseAnnotations(raw: string): Record<string, string> {
  if (!raw) {
    return {}
  }
  try {
    const parsed: unknown = JSON.parse(raw)
    if (typeof parsed === 'object' && parsed !== null && !Array.isArray(parsed)) {
      const result: Record<string, string> = {}
      for (const [key, value] of Object.entries(parsed as Record<string, unknown>)) {
        result[key] = String(value)
      }
      return result
    }
    return {}
  } catch {
    return {}
  }
}

function parseJsonArray(raw: string): unknown[] {
  if (!raw) return []
  try {
    const parsed: unknown = JSON.parse(raw)
    return Array.isArray(parsed) ? parsed : []
  } catch {
    return []
  }
}

function parseJsonObject(raw: string): Record<string, string> {
  if (!raw) return {}
  try {
    const parsed: unknown = JSON.parse(raw)
    if (typeof parsed === 'object' && parsed !== null && !Array.isArray(parsed)) {
      const result: Record<string, string> = {}
      for (const [key, value] of Object.entries(parsed as Record<string, unknown>)) {
        result[key] = String(value)
      }
      return result
    }
    return {}
  } catch {
    return {}
  }
}

const VALID_REPO_STATUSES: ReadonlySet<string> = new Set(['Cloning', 'Ready', 'Failed'])
const VALID_CONDITION_STATUSES: ReadonlySet<string> = new Set(['True', 'False', 'Unknown'])

function parseRepos(raw: string): DomainRepo[] {
  return parseJsonArray(raw).map((item) => {
    const r = item as Record<string, unknown>
    return {
      url: String(r.url ?? ''),
      branch: r.branch ? String(r.branch) : null,
      name: r.name ? String(r.name) : null,
      autoPush: Boolean(r.autoPush),
    }
  })
}

function parseReconciledRepos(raw: string): DomainReconciledRepo[] {
  return parseJsonArray(raw).map((item) => {
    const r = item as Record<string, unknown>
    const status = String(r.status ?? '')
    return {
      url: String(r.url ?? ''),
      name: r.name ? String(r.name) : null,
      status: VALID_REPO_STATUSES.has(status) ? (status as ReconciledRepoStatus) : null,
      currentActiveBranch: r.currentActiveBranch ? String(r.currentActiveBranch) : null,
      defaultBranch: r.defaultBranch ? String(r.defaultBranch) : null,
      clonedAt: r.clonedAt ? String(r.clonedAt) : null,
    }
  })
}

function parseConditions(raw: string): DomainCondition[] {
  return parseJsonArray(raw).map((item) => {
    const c = item as Record<string, unknown>
    const status = String(c.status ?? 'Unknown')
    return {
      type: String(c.type ?? ''),
      status: VALID_CONDITION_STATUSES.has(status) ? (status as ConditionStatus) : 'Unknown',
      reason: c.reason ? String(c.reason) : null,
      message: c.message ? String(c.message) : null,
      lastTransitionTime: c.lastTransitionTime ? String(c.lastTransitionTime) : null,
    }
  })
}

function emptyToNull(value: string): string | null {
  return value || null
}

function numberOrNull(value: number): number | null {
  return value === 0 || value === undefined || value === null ? null : value
}

export function mapSdkSessionToDomain(sdk: Session): DomainSession {
  const annotations = parseAnnotations(sdk.annotations)
  return {
    id: sdk.id,
    name: sdk.name,
    phase: parsePhase(sdk.phase),
    agentId: emptyToNull(sdk.agent_id),
    agentName: annotations['agent_name'] ?? null,
    projectId: emptyToNull(sdk.project_id),
    model: emptyToNull(sdk.llm_model),
    temperature: numberOrNull(sdk.llm_temperature),
    maxTokens: numberOrNull(sdk.llm_max_tokens),
    timeout: numberOrNull(sdk.timeout),
    workflowId: emptyToNull(sdk.workflow_id),
    prompt: emptyToNull(sdk.prompt),
    sdkRestartCount: sdk.sdk_restart_count ?? 0,
    startTime: emptyToNull(sdk.start_time),
    completionTime: emptyToNull(sdk.completion_time),
    createdAt: sdk.created_at ?? '',
    updatedAt: sdk.updated_at ?? '',
    annotations,
    labels: parseJsonObject(sdk.labels),
    environmentVariables: parseJsonObject(sdk.environment_variables),
    repos: parseRepos(sdk.repos),
    reconciledRepos: parseReconciledRepos(sdk.reconciled_repos),
    conditions: parseConditions(sdk.conditions),
  }
}

export function mapSdkProjectToDomain(sdk: Project): DomainProject {
  return {
    id: sdk.id,
    name: sdk.name,
    description: emptyToNull(sdk.description),
    status: emptyToNull(sdk.status),
    createdAt: sdk.created_at ?? '',
    updatedAt: sdk.updated_at ?? '',
  }
}

export type SdkSessionMessageShape = {
  id: string
  session_id: string
  event_type: string
  payload: string
  seq: number
  created_at: string | null
}

const VALID_EVENT_TYPES: ReadonlySet<string> = new Set<string>([
  'user', 'assistant', 'text', 'tool_use', 'tool_result',
  'error', 'lifecycle', 'user_feedback', 'system',
])

function parseEventType(raw: string): SessionEventType {
  if (VALID_EVENT_TYPES.has(raw)) {
    return raw as SessionEventType
  }
  return 'system'
}

export function mapSessionMessageToDomain(sdk: SdkSessionMessageShape): DomainSessionMessage {
  return {
    id: sdk.id,
    sessionId: sdk.session_id,
    eventType: parseEventType(sdk.event_type),
    payload: sdk.payload,
    seq: sdk.seq,
    createdAt: sdk.created_at ?? '',
  }
}
