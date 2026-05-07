import type { ScheduledSession as SdkScheduledSession, Agent as SdkAgent } from '@/lib/sdk';
import type { ScheduledSessionsPort } from '../../ports/scheduled-sessions';
import type { ScheduledSession, CreateScheduledSessionRequest } from '@/types/api/scheduled-sessions';
import type { CreateAgenticSessionRequest } from '@/types/api/sessions';
import type { AgenticSession } from '@/types/api/sessions';
import { getClient } from './client';
import { parseJsonField } from './json';
import { wrapSdkError } from './errors';

function agentToSessionTemplate(agent: SdkAgent): CreateAgenticSessionRequest {
  return {
    initialPrompt: agent.prompt || undefined,
    llmSettings: {
      model: agent.llm_model || '',
      temperature: agent.llm_temperature ?? 0,
      maxTokens: agent.llm_max_tokens ?? 0,
    },
    displayName: agent.display_name || undefined,
    environmentVariables: parseJsonField<Record<string, string>>(agent.environment_variables, {}),
    repos: agent.repo_url ? [{ url: agent.repo_url }] : undefined,
    activeWorkflow: agent.workflow_id
      ? { gitUrl: agent.workflow_id, branch: 'main' }
      : undefined,
    labels: parseJsonField<Record<string, string>>(agent.labels, {}),
    annotations: parseJsonField<Record<string, string>>(agent.annotations, {}),
  };
}

function toScheduledSession(
  s: SdkScheduledSession,
  agent: SdkAgent | undefined,
): ScheduledSession {
  const sessionTemplate = agent
    ? agentToSessionTemplate(agent)
    : { initialPrompt: s.session_prompt || undefined };

  return {
    name: s.name,
    namespace: '',
    creationTimestamp: s.created_at || '',
    schedule: s.schedule,
    suspend: !s.enabled,
    displayName: s.name,
    sessionTemplate,
    lastScheduleTime: s.last_run_at || undefined,
    activeCount: 0,
    reuseLastSession: s.stop_on_run_finished,
  };
}

async function resolveAgents(
  projectName: string,
  scheduledSessions: SdkScheduledSession[],
): Promise<Map<string, SdkAgent>> {
  const agentIds = [...new Set(scheduledSessions.map(s => s.agent_id).filter(Boolean))];
  if (agentIds.length === 0) return new Map();

  const client = getClient(projectName);
  const agents = new Map<string, SdkAgent>();

  const results = await Promise.allSettled(
    agentIds.map(id => client.agents.get(id)),
  );
  for (const result of results) {
    if (result.status === 'fulfilled') {
      agents.set(result.value.id, result.value);
    }
  }
  return agents;
}

function sessionTemplateToAgentRequest(
  projectName: string,
  template: CreateAgenticSessionRequest,
  displayName?: string,
): {
  name: string;
  project_id: string;
  display_name?: string;
  prompt?: string;
  llm_model?: string;
  llm_temperature?: number;
  llm_max_tokens?: number;
  repo_url?: string;
  workflow_id?: string;
  environment_variables?: string;
  labels?: string;
  annotations?: string;
} {
  return {
    name: displayName || `agent-${Date.now()}`,
    project_id: projectName,
    display_name: displayName,
    prompt: template.initialPrompt,
    llm_model: template.llmSettings?.model,
    llm_temperature: template.llmSettings?.temperature,
    llm_max_tokens: template.llmSettings?.maxTokens,
    repo_url: template.repos?.[0]?.url,
    workflow_id: template.activeWorkflow?.gitUrl,
    environment_variables: template.environmentVariables ? JSON.stringify(template.environmentVariables) : undefined,
    labels: template.labels ? JSON.stringify(template.labels) : undefined,
    annotations: template.annotations ? JSON.stringify(template.annotations) : undefined,
  };
}

export function createScheduledSessionsAdapter(): ScheduledSessionsPort {
  return {
    async listScheduledSessions(projectName: string) {
      try {
        const client = getClient(projectName);
        const list = await client.scheduledSessions.list({ size: 1000 });
        const agents = await resolveAgents(projectName, list.items);
        return list.items.map(s => toScheduledSession(s, agents.get(s.agent_id)));
      } catch (err) {
        wrapSdkError(err);
      }
    },

    async getScheduledSession(projectName: string, name: string) {
      try {
        const client = getClient(projectName);
        const sdk = await client.scheduledSessions.get(name);
        let agent: SdkAgent | undefined;
        if (sdk.agent_id) {
          try {
            agent = await client.agents.get(sdk.agent_id);
          } catch {
            // Agent may have been deleted
          }
        }
        return toScheduledSession(sdk, agent);
      } catch (err) {
        wrapSdkError(err);
      }
    },

    async createScheduledSession(projectName: string, data: CreateScheduledSessionRequest) {
      try {
        const client = getClient(projectName);
        const agentReq = sessionTemplateToAgentRequest(projectName, data.sessionTemplate, data.displayName);
        const agent = await client.agents.create(agentReq);

        const sdk = await client.scheduledSessions.create({
          name: data.displayName || `scheduled-${Date.now()}`,
          project_id: projectName,
          schedule: data.schedule,
          agent_id: agent.id,
          enabled: data.suspend !== true,
          session_prompt: data.sessionTemplate.initialPrompt,
          stop_on_run_finished: data.reuseLastSession,
          timeout: data.sessionTemplate.timeout,
        });
        return toScheduledSession(sdk, agent);
      } catch (err) {
        wrapSdkError(err);
      }
    },

    async updateScheduledSession(projectName: string, name: string, data) {
      try {
        const client = getClient(projectName);
        const existing = await client.scheduledSessions.get(name);

        if (data.sessionTemplate && existing.agent_id) {
          const agentPatch: Record<string, unknown> = {};
          const t = data.sessionTemplate;
          if (t.initialPrompt !== undefined) agentPatch.prompt = t.initialPrompt;
          if (t.llmSettings?.model !== undefined) agentPatch.llm_model = t.llmSettings.model;
          if (t.llmSettings?.temperature !== undefined) agentPatch.llm_temperature = t.llmSettings.temperature;
          if (t.llmSettings?.maxTokens !== undefined) agentPatch.llm_max_tokens = t.llmSettings.maxTokens;
          if (t.repos?.[0]?.url !== undefined) agentPatch.repo_url = t.repos[0].url;
          if (t.activeWorkflow?.gitUrl !== undefined) agentPatch.workflow_id = t.activeWorkflow.gitUrl;
          if (t.displayName !== undefined) agentPatch.display_name = t.displayName;

          if (Object.keys(agentPatch).length > 0) {
            await client.agents.update(existing.agent_id, agentPatch);
          }
        }

        const ssPatch: Record<string, unknown> = {};
        if (data.schedule !== undefined) ssPatch.schedule = data.schedule;
        if (data.suspend !== undefined) ssPatch.enabled = !data.suspend;
        if (data.displayName !== undefined) ssPatch.name = data.displayName;
        if (data.reuseLastSession !== undefined) ssPatch.stop_on_run_finished = data.reuseLastSession;

        const sdk = Object.keys(ssPatch).length > 0
          ? await client.scheduledSessions.update(name, ssPatch)
          : existing;

        let agent: SdkAgent | undefined;
        if (sdk.agent_id) {
          try {
            agent = await client.agents.get(sdk.agent_id);
          } catch {
            // Agent may have been deleted
          }
        }
        return toScheduledSession(sdk, agent);
      } catch (err) {
        wrapSdkError(err);
      }
    },

    async deleteScheduledSession(projectName: string, name: string) {
      try {
        const client = getClient(projectName);
        const existing = await client.scheduledSessions.get(name);
        await client.scheduledSessions.delete(name);
        if (existing.agent_id) {
          try {
            await client.agents.delete(existing.agent_id);
          } catch {
            // Agent may already be deleted or used elsewhere
          }
        }
      } catch (err) {
        wrapSdkError(err);
      }
    },

    async suspendScheduledSession(projectName: string, name: string) {
      try {
        const client = getClient(projectName);
        const sdk = await client.scheduledSessions.suspend(name);
        let agent: SdkAgent | undefined;
        if (sdk.agent_id) {
          try {
            agent = await client.agents.get(sdk.agent_id);
          } catch { /* ignore */ }
        }
        return toScheduledSession(sdk, agent);
      } catch (err) {
        wrapSdkError(err);
      }
    },

    async resumeScheduledSession(projectName: string, name: string) {
      try {
        const client = getClient(projectName);
        const sdk = await client.scheduledSessions.resume(name);
        let agent: SdkAgent | undefined;
        if (sdk.agent_id) {
          try {
            agent = await client.agents.get(sdk.agent_id);
          } catch { /* ignore */ }
        }
        return toScheduledSession(sdk, agent);
      } catch (err) {
        wrapSdkError(err);
      }
    },

    async triggerScheduledSession(projectName: string, name: string) {
      try {
        const client = getClient(projectName);
        const result = await client.scheduledSessions.trigger(name);
        return {
          name: (result as Record<string, string>).name || name,
          namespace: (result as Record<string, string>).namespace || '',
        };
      } catch (err) {
        wrapSdkError(err);
      }
    },

    async listScheduledSessionRuns(projectName: string, name: string) {
      try {
        const client = getClient(projectName);
        const result = await client.scheduledSessions.runs(name);
        return ((result as Record<string, unknown>).items as AgenticSession[]) || [];
      } catch (err) {
        wrapSdkError(err);
      }
    },
  };
}

export const scheduledSessionsAdapter = createScheduledSessionsAdapter();
