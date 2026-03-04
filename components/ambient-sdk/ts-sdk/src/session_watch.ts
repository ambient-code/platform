import type { Session } from './session';
import type { AmbientClientConfig } from './base';

export type SessionWatchEventType = 'CREATED' | 'UPDATED' | 'DELETED';

export type SessionWatchEvent = {
  type: SessionWatchEventType;
  session?: Session;
  resourceId: string;
};

export type WatchOptions = {
  resourceVersion?: string;
  timeout?: number;
  signal?: AbortSignal;
};

type RawSessionPayload = Record<string, unknown>;

export class SessionWatcher {
  private readonly config: AmbientClientConfig;
  private readonly opts: WatchOptions;
  private controller: AbortController | null = null;
  private closed = false;

  constructor(config: AmbientClientConfig, opts: WatchOptions = {}) {
    this.config = config;
    this.opts = opts;
  }

  async *watch(): AsyncGenerator<SessionWatchEvent, void, unknown> {
    if (this.closed) {
      throw new Error('Watcher is closed');
    }

    this.controller = new AbortController();
    const internalSignal = this.controller.signal;

    if (this.opts.signal) {
      this.opts.signal.addEventListener('abort', () => this.close());
    }

    let timeoutId: ReturnType<typeof setTimeout> | null = null;
    if (this.opts.timeout) {
      timeoutId = setTimeout(() => this.close(), this.opts.timeout);
    }

    try {
      const url = this.buildWatchURL();
      const response = await fetch(url, {
        method: 'GET',
        headers: {
          'Authorization': `Bearer ${this.config.token}`,
          'X-Ambient-Project': this.config.project,
          'Accept': 'text/event-stream',
          'Cache-Control': 'no-cache',
        },
        signal: internalSignal,
      });

      if (!response.ok) {
        throw new Error(`Watch stream HTTP error: ${response.status} ${response.statusText}`);
      }

      if (!response.body) {
        throw new Error('Watch stream returned no body');
      }

      const reader = response.body.getReader();
      const decoder = new TextDecoder();
      let buffer = '';

      try {
        while (!this.closed) {
          const { done, value } = await reader.read();
          if (done) break;

          buffer += decoder.decode(value, { stream: true });

          const lines = buffer.split('\n');
          buffer = lines.pop() || '';

          for (const line of lines) {
            const trimmed = line.trim();
            if (!trimmed || !trimmed.startsWith('data: ')) continue;

            const dataStr = trimmed.slice(6);
            if (dataStr === '[DONE]') return;

            const event = this.parseEventData(dataStr);
            if (event) {
              yield event;
            }
          }
        }
      } finally {
        reader.cancel().catch(() => {});
      }
    } catch (error) {
      if (this.closed) return;
      throw error;
    } finally {
      if (timeoutId) clearTimeout(timeoutId);
      this.close();
    }
  }

  close(): void {
    this.closed = true;
    if (this.controller) {
      this.controller.abort();
      this.controller = null;
    }
  }

  private buildWatchURL(): string {
    const baseUrl = this.config.baseUrl.replace(/\/+$/, '');
    const path = `${baseUrl}/api/ambient/v1/sessions?watch=true`;
    if (this.opts.resourceVersion) {
      return `${path}&resourceVersion=${encodeURIComponent(this.opts.resourceVersion)}`;
    }
    return path;
  }

  private parseEventData(data: string): SessionWatchEvent | null {
    if (!data || data === '[DONE]') {
      return null;
    }

    try {
      const parsed: unknown = JSON.parse(data);

      if (!parsed || typeof parsed !== 'object') {
        return null;
      }

      const record = parsed as Record<string, unknown>;

      return {
        type: (typeof record.type === 'string' ? record.type : 'UNKNOWN') as SessionWatchEventType,
        session: record.object && typeof record.object === 'object'
          ? this.convertSession(record.object as RawSessionPayload)
          : undefined,
        resourceId: typeof record.resource_id === 'string' ? record.resource_id : '',
      };
    } catch {
      return null;
    }
  }

  private convertSession(raw: RawSessionPayload): Session {
    return {
      id: String(raw.id ?? ''),
      kind: String(raw.kind ?? 'Session'),
      href: String(raw.href ?? ''),
      created_at: raw.created_at != null ? String(raw.created_at) : null,
      updated_at: raw.updated_at != null ? String(raw.updated_at) : null,
      name: String(raw.name ?? ''),
      repo_url: String(raw.repo_url ?? ''),
      prompt: String(raw.prompt ?? ''),
      created_by_user_id: String(raw.created_by_user_id ?? ''),
      assigned_user_id: String(raw.assigned_user_id ?? ''),
      workflow_id: String(raw.workflow_id ?? ''),
      repos: String(raw.repos ?? ''),
      timeout: typeof raw.timeout === 'number' ? raw.timeout : 0,
      llm_model: String(raw.llm_model ?? ''),
      llm_temperature: typeof raw.llm_temperature === 'number' ? raw.llm_temperature : 0,
      llm_max_tokens: typeof raw.llm_max_tokens === 'number' ? raw.llm_max_tokens : 0,
      parent_session_id: String(raw.parent_session_id ?? ''),
      bot_account_name: String(raw.bot_account_name ?? ''),
      resource_overrides: String(raw.resource_overrides ?? ''),
      environment_variables: String(raw.environment_variables ?? ''),
      labels: String(raw.labels ?? ''),
      annotations: String(raw.annotations ?? ''),
      project_id: String(raw.project_id ?? ''),
      phase: String(raw.phase ?? ''),
      start_time: String(raw.start_time ?? ''),
      completion_time: String(raw.completion_time ?? ''),
      sdk_session_id: String(raw.sdk_session_id ?? ''),
      sdk_restart_count: typeof raw.sdk_restart_count === 'number' ? raw.sdk_restart_count : 0,
      conditions: String(raw.conditions ?? ''),
      reconciled_repos: String(raw.reconciled_repos ?? ''),
      reconciled_workflow: String(raw.reconciled_workflow ?? ''),
      kube_cr_name: String(raw.kube_cr_name ?? ''),
      kube_cr_uid: String(raw.kube_cr_uid ?? ''),
      kube_namespace: String(raw.kube_namespace ?? ''),
    };
  }
}

export const SessionWatchEventUtils = {
  isCreated: (event: SessionWatchEvent): boolean => event.type === 'CREATED',
  isUpdated: (event: SessionWatchEvent): boolean => event.type === 'UPDATED',
  isDeleted: (event: SessionWatchEvent): boolean => event.type === 'DELETED',
};
