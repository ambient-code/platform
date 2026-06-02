import type { Session, Project } from 'ambient-sdk'
import type { DomainSession, DomainProject, DomainSessionMessage, SessionPhase, SessionEventType } from '@/domain/types'

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

function emptyToNull(value: string): string | null {
  return value || null
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
    startTime: emptyToNull(sdk.start_time),
    completionTime: emptyToNull(sdk.completion_time),
    createdAt: sdk.created_at ?? '',
    updatedAt: sdk.updated_at ?? '',
    annotations,
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
