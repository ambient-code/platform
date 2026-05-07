import type { Session as SdkSession } from '@/lib/sdk';
import type { SessionsPort } from '../../ports/sessions';
import type { AgenticSession, AgenticSessionPhase, LLMSettings, SessionRepo, ReconciledRepo, ReconciledWorkflow, SessionCondition, ResourceOverrides } from '@/types/api/sessions';
import type { PaginationParams } from '@/types/api/common';
import { ApiClientError } from '@/types/api/common';
import * as sessionsApi from '../../api/sessions';
import { getClient } from './client';
import { toListOptions, fromSdkList } from './pagination';
import { parseJsonField } from './json';
import { wrapSdkError } from './errors';

function toSession(s: SdkSession): AgenticSession {
  const labels = parseJsonField<Record<string, string>>(s.labels, {});
  const annotations = parseJsonField<Record<string, string>>(s.annotations, {});
  const repos = parseJsonField<SessionRepo[]>(s.repos, []);
  const envVars = parseJsonField<Record<string, string>>(s.environment_variables, {});
  const conditions = parseJsonField<SessionCondition[]>(s.conditions, []);
  const reconciledRepos = parseJsonField<ReconciledRepo[]>(s.reconciled_repos, []);
  const reconciledWorkflow = parseJsonField<ReconciledWorkflow | undefined>(s.reconciled_workflow, undefined);
  const resourceOverrides = parseJsonField<ResourceOverrides | undefined>(s.resource_overrides, undefined);

  const llmSettings: LLMSettings = {
    model: s.llm_model || '',
    temperature: s.llm_temperature ?? 0,
    maxTokens: s.llm_max_tokens ?? 0,
  };

  return {
    metadata: {
      name: s.kube_cr_name || s.name,
      namespace: s.kube_namespace || '',
      creationTimestamp: s.created_at || '',
      uid: s.kube_cr_uid || s.id,
      labels: Object.keys(labels).length > 0 ? labels : undefined,
      annotations: Object.keys(annotations).length > 0 ? annotations : undefined,
    },
    spec: {
      initialPrompt: s.prompt || undefined,
      llmSettings,
      timeout: s.timeout ?? 0,
      displayName: s.name,
      environmentVariables: Object.keys(envVars).length > 0 ? envVars : undefined,
      repos: repos.length > 0 ? repos : undefined,
      activeWorkflow: s.workflow_id
        ? { gitUrl: s.workflow_id, branch: 'main' }
        : undefined,
    },
    status: {
      phase: (s.phase as AgenticSessionPhase) || 'Pending',
      startTime: s.start_time || undefined,
      completionTime: s.completion_time || undefined,
      reconciledRepos: reconciledRepos.length > 0 ? reconciledRepos : undefined,
      reconciledWorkflow,
      sdkSessionId: s.sdk_session_id || undefined,
      sdkRestartCount: s.sdk_restart_count || undefined,
      conditions: conditions.length > 0 ? conditions : undefined,
    },
    ...(resourceOverrides ? {} : {}),
  };
}

export function createSessionsAdapter(): SessionsPort {
  return {
    async listSessions(projectName: string, params?: PaginationParams) {
      try {
        const client = getClient(projectName);
        const opts = toListOptions(params);
        const list = await client.sessions.list(opts);
        return fromSdkList(list, toSession, (p) => this.listSessions(projectName, p));
      } catch (err) {
        wrapSdkError(err);
      }
    },

    async getSession(projectName: string, sessionName: string) {
      try {
        const client = getClient(projectName);
        const sdk = await client.sessions.get(sessionName);
        return toSession(sdk);
      } catch (err) {
        wrapSdkError(err);
      }
    },

    async createSession(projectName: string, data) {
      try {
        const client = getClient(projectName);
        const sdk = await client.sessions.create({
          name: data.displayName || `session-${Date.now()}`,
          prompt: data.initialPrompt,
          llm_model: data.llmSettings?.model,
          llm_temperature: data.llmSettings?.temperature,
          llm_max_tokens: data.llmSettings?.maxTokens,
          timeout: data.timeout,
          labels: data.labels ? JSON.stringify(data.labels) : undefined,
          annotations: data.annotations ? JSON.stringify(data.annotations) : undefined,
          environment_variables: data.environmentVariables ? JSON.stringify(data.environmentVariables) : undefined,
          repos: data.repos ? JSON.stringify(data.repos) : undefined,
          workflow_id: data.activeWorkflow?.gitUrl,
          parent_session_id: data.parent_session_id,
        });
        return toSession(sdk);
      } catch (err) {
        wrapSdkError(err);
      }
    },

    async stopSession(projectName: string, sessionName: string) {
      try {
        const client = getClient(projectName);
        await client.sessions.stop(sessionName);
        return 'Session stopped';
      } catch (err) {
        wrapSdkError(err);
      }
    },

    async startSession(projectName: string, sessionName: string) {
      try {
        const client = getClient(projectName);
        await client.sessions.start(sessionName);
        return { message: 'Session started' };
      } catch (err) {
        wrapSdkError(err);
      }
    },

    async cloneSession(projectName, sessionName, data) {
      return sessionsApi.cloneSession(projectName, sessionName, data);
    },

    async deleteSession(projectName: string, sessionName: string) {
      try {
        const client = getClient(projectName);
        await client.sessions.delete(sessionName);
      } catch (err) {
        wrapSdkError(err);
      }
    },

    async getSessionPodEvents(projectName, sessionName) {
      return sessionsApi.getSessionPodEvents(projectName, sessionName);
    },

    async updateSessionDisplayName(projectName: string, sessionName: string, displayName: string) {
      try {
        const client = getClient(projectName);
        const sdk = await client.sessions.update(sessionName, { name: displayName });
        return toSession(sdk);
      } catch (err) {
        wrapSdkError(err);
      }
    },

    async getSessionExport(projectName, sessionName) {
      return sessionsApi.getSessionExport(projectName, sessionName);
    },

    async switchSessionModel(projectName: string, sessionName: string, model: string) {
      try {
        const client = getClient(projectName);
        const sdk = await client.sessions.update(sessionName, { llm_model: model });
        return toSession(sdk);
      } catch (err) {
        wrapSdkError(err);
      }
    },

    async saveToGoogleDrive() {
      throw new ApiClientError('Not implemented in v2 adapter', 'NOT_IMPLEMENTED');
    },
  };
}

export const sessionsAdapter = createSessionsAdapter();
