import type { Session } from '@ambient-platform/sdk';
import type {
  AgenticSession,
  AgenticSessionPhase,
  SessionRepo,
  ReconciledRepo,
  ReconciledWorkflow,
  SessionCondition,
} from '@/types/agentic-session';

function parseJsonField<T>(value: string | null | undefined, fallback: T): T {
  if (!value) return fallback;
  try {
    return JSON.parse(value) as T;
  } catch {
    return fallback;
  }
}

export function v1SessionToAgenticSession(s: Session): AgenticSession {
  const repos = parseJsonField<SessionRepo[]>(s.repos, []);
  const reconciledRepos = parseJsonField<ReconciledRepo[] | undefined>(s.reconciled_repos, undefined);
  const reconciledWorkflow = parseJsonField<ReconciledWorkflow | undefined>(s.reconciled_workflow, undefined);
  const conditions = parseJsonField<SessionCondition[]>(s.conditions, []);
  const labels = parseJsonField<Record<string, string>>(s.labels, {});
  const annotations = parseJsonField<Record<string, string>>(s.annotations, {});

  return {
    metadata: {
      name: s.kube_cr_name || s.name || s.id,
      namespace: s.kube_namespace || '',
      creationTimestamp: s.created_at || '',
      uid: s.kube_cr_uid || s.id,
      labels,
      annotations,
    },
    spec: {
      initialPrompt: s.prompt || undefined,
      llmSettings: {
        model: s.llm_model || 'sonnet',
        temperature: s.llm_temperature ?? 0.7,
        maxTokens: s.llm_max_tokens ?? 4000,
      },
      timeout: s.timeout ?? 3600,
      displayName: s.name || undefined,
      project: s.project_id || undefined,
      interactive: s.interactive,
      repos: repos.length > 0 ? repos : undefined,
    },
    status: {
      phase: (s.phase as AgenticSessionPhase) || 'Pending',
      startTime: s.start_time || undefined,
      completionTime: s.completion_time || undefined,
      reconciledRepos: reconciledRepos,
      reconciledWorkflow,
      sdkSessionId: s.sdk_session_id || undefined,
      sdkRestartCount: s.sdk_restart_count ?? undefined,
      conditions: conditions.length > 0 ? conditions : undefined,
    },
  };
}
