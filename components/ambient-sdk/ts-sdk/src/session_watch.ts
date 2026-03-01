// Session watch functionality for real-time streaming of session changes

import type { Session } from './session';
import type { AmbientClientConfig } from './base';

/**
 * Represents a real-time session change event
 */
export interface SessionWatchEvent {
  /** Type of the watch event */
  type: 'CREATED' | 'UPDATED' | 'DELETED';
  /** Session object that changed (may be null for DELETED events) */
  session?: Session;
  /** ID of the resource that changed */
  resourceId: string;
}

/**
 * Options for configuring session watching
 */
export interface WatchOptions {
  /** Resource version to start watching from */
  resourceVersion?: string;
  /** Timeout in milliseconds */
  timeout?: number;
  /** Signal to abort the watch stream */
  signal?: AbortSignal;
}

/**
 * Session watcher that provides real-time event streaming
 */
export class SessionWatcher {
  private readonly config: AmbientClientConfig;
  private eventSource: EventSource | null = null;
  private closed = false;

  constructor(config: AmbientClientConfig) {
    this.config = config;
  }

  /**
   * Start watching for session events using Server-Sent Events
   */
  async *watch(opts: WatchOptions = {}): AsyncGenerator<SessionWatchEvent, void, unknown> {
    if (this.closed) {
      throw new Error('Watcher is closed');
    }

    const url = this.buildWatchURL();
    
    try {
      // Use EventSource for SSE streaming
      this.eventSource = new EventSource(url);
      
      // Set up event handlers
      const eventQueue: SessionWatchEvent[] = [];
      const errors: Error[] = [];
      let finished = false;

      this.eventSource.onmessage = (event) => {
        try {
          const eventData = this.parseEventData(event.data);
          if (eventData) {
            eventQueue.push(eventData);
          }
        } catch (error) {
          errors.push(error instanceof Error ? error : new Error(String(error)));
        }
      };

      this.eventSource.onerror = (error) => {
        errors.push(new Error('EventSource error'));
        finished = true;
      };

      // Handle timeout
      const timeoutId = opts.timeout 
        ? setTimeout(() => {
            this.close();
            finished = true;
          }, opts.timeout)
        : null;

      // Handle abort signal
      const abortHandler = () => {
        this.close();
        finished = true;
      };
      opts.signal?.addEventListener('abort', abortHandler);

      try {
        // Yield events as they arrive
        while (!finished && !this.closed) {
          if (errors.length > 0) {
            throw errors[0];
          }

          if (eventQueue.length > 0) {
            yield eventQueue.shift()!;
          } else {
            // Wait a bit before checking again
            await new Promise(resolve => setTimeout(resolve, 10));
          }
        }
      } finally {
        if (timeoutId) clearTimeout(timeoutId);
        opts.signal?.removeEventListener('abort', abortHandler);
        this.close();
      }

    } catch (error) {
      this.close();
      throw error;
    }
  }

  /**
   * Close the watcher and clean up resources
   */
  close(): void {
    this.closed = true;
    if (this.eventSource) {
      this.eventSource.close();
      this.eventSource = null;
    }
  }

  /**
   * Build the SSE watch URL
   */
  private buildWatchURL(): string {
    const baseUrl = this.config.baseUrl.replace(/\/+$/, '');
    const url = new URL(`${baseUrl}/api/ambient/v1/sessions`, window.location.href);
    url.searchParams.set('watch', 'true');
    return url.toString();
  }

  /**
   * Parse EventSource event data into SessionWatchEvent
   */
  private parseEventData(data: string): SessionWatchEvent | null {
    if (!data || data === '[DONE]') {
      return null;
    }

    try {
      const parsed = JSON.parse(data);
      
      if (!parsed || typeof parsed !== 'object') {
        return null;
      }

      return {
        type: parsed.type || 'UNKNOWN',
        session: parsed.object ? this.convertSession(parsed.object) : undefined,
        resourceId: parsed.resource_id || '',
      };
    } catch {
      return null;
    }
  }

  /**
   * Convert raw session object to typed Session
   */
  private convertSession(raw: any): Session {
    // Basic conversion matching the generated Session type with snake_case fields
    return {
      id: raw.id || '',
      kind: raw.kind || 'Session',
      href: raw.href || '',
      created_at: raw.created_at || '',
      updated_at: raw.updated_at || '',
      name: raw.name || '',
      repo_url: raw.repo_url || '',
      prompt: raw.prompt || '',
      created_by_user_id: raw.created_by_user_id || '',
      assigned_user_id: raw.assigned_user_id || '',
      workflow_id: raw.workflow_id || '',
      repos: raw.repos || '',
      timeout: raw.timeout || 0,
      llm_model: raw.llm_model || '',
      llm_temperature: raw.llm_temperature || 0,
      llm_max_tokens: raw.llm_max_tokens || 0,
      parent_session_id: raw.parent_session_id || '',
      bot_account_name: raw.bot_account_name || '',
      resource_overrides: raw.resource_overrides || '',
      environment_variables: raw.environment_variables || '',
      labels: raw.labels || '',
      annotations: raw.annotations || '',
      project_id: raw.project_id || '',
      phase: raw.phase || '',
      start_time: raw.start_time || '',
      completion_time: raw.completion_time || '',
      sdk_session_id: raw.sdk_session_id || '',
      sdk_restart_count: raw.sdk_restart_count || 0,
      conditions: raw.conditions || '',
      reconciled_repos: raw.reconciled_repos || '',
      reconciled_workflow: raw.reconciled_workflow || '',
      kube_cr_name: raw.kube_cr_name || '',
      kube_cr_uid: raw.kube_cr_uid || '',
      kube_namespace: raw.kube_namespace || '',
    };
  }
}

/**
 * Utility functions for checking event types
 */
export const SessionWatchEventUtils = {
  isCreated: (event: SessionWatchEvent): boolean => event.type === 'CREATED',
  isUpdated: (event: SessionWatchEvent): boolean => event.type === 'UPDATED',
  isDeleted: (event: SessionWatchEvent): boolean => event.type === 'DELETED',
};