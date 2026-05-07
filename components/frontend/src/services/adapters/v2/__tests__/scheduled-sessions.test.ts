import { describe, it, expect } from 'vitest';
import { createScheduledSessionsAdapter } from '../scheduled-sessions';
import { ApiClientError } from '@/types/api/common';
import type { SdkClient } from '../client';
import type { ScheduledSession as SdkScheduledSession, Agent as SdkAgent } from '@/lib/sdk';

const sdkScheduledSession: SdkScheduledSession = {
  id: 'ss-uuid',
  kind: 'ScheduledSession',
  href: '/api/ambient/v1/scheduled-sessions/ss-uuid',
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-02T00:00:00Z',
  name: 'daily-build',
  schedule: '0 9 * * *',
  enabled: true,
  agent_id: 'agent-1',
  session_prompt: 'Run tests',
  last_run_at: '2026-01-02T09:00:00Z',
  next_run_at: '2026-01-03T09:00:00Z',
  project_id: 'proj-1',
  stop_on_run_finished: false,
  description: '',
  inactivity_timeout: 0,
  runner_type: '',
  timeout: 0,
  timezone: '',
};

const sdkAgent: SdkAgent = {
  id: 'agent-1',
  kind: 'Agent',
  href: '/api/ambient/v1/agents/agent-1',
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
  name: 'build-agent',
  display_name: 'Build Agent',
  prompt: 'Run the build',
  llm_model: 'claude-sonnet-4-5',
  llm_temperature: 0.5,
  llm_max_tokens: 4096,
  repo_url: 'https://github.com/org/repo',
  workflow_id: 'wf-1',
  environment_variables: '{"CI":"true"}',
  labels: '{}',
  annotations: '{}',
  project_id: 'proj-1',
  description: '',
  bot_account_name: '',
  current_session_id: '',
  owner_user_id: '',
  parent_agent_id: '',
  resource_overrides: '',
};

type FakeOverrides = {
  scheduledSessions?: Partial<{
    list: (opts?: unknown) => Promise<{ items: SdkScheduledSession[]; total: number; page: number; size: number }>;
    get: (id: string) => Promise<SdkScheduledSession>;
    create: (data: unknown) => Promise<SdkScheduledSession>;
    update: (id: string, data: unknown) => Promise<SdkScheduledSession>;
    delete: (id: string) => Promise<void>;
    suspend: (id: string) => Promise<SdkScheduledSession>;
    resume: (id: string) => Promise<SdkScheduledSession>;
    trigger: (id: string) => Promise<Record<string, unknown>>;
    runs: (id: string) => Promise<Record<string, unknown>>;
  }>;
  agents?: Partial<{
    get: (id: string) => Promise<SdkAgent>;
    create: (data: unknown) => Promise<SdkAgent>;
    update: (id: string, data: unknown) => Promise<SdkAgent>;
    delete: (id: string) => Promise<void>;
  }>;
};

function makeFakeClient(overrides?: FakeOverrides): SdkClient {
  const ss = overrides?.scheduledSessions ?? {};
  const ag = overrides?.agents ?? {};
  return {
    projects: {} as SdkClient['projects'],
    sessions: {} as SdkClient['sessions'],
    scheduledSessions: {
      list: ss.list ?? (async () => ({ kind: 'ScheduledSessionList', items: [sdkScheduledSession], total: 1, page: 1, size: 1000 })),
      get: ss.get ?? (async () => sdkScheduledSession),
      create: ss.create ?? (async () => sdkScheduledSession),
      update: ss.update ?? (async () => sdkScheduledSession),
      delete: ss.delete ?? (async () => undefined),
      suspend: ss.suspend ?? (async () => ({ ...sdkScheduledSession, enabled: false })),
      resume: ss.resume ?? (async () => ({ ...sdkScheduledSession, enabled: true })),
      trigger: ss.trigger ?? (async () => ({ name: 'triggered-run', namespace: 'proj-ns' })),
      runs: ss.runs ?? (async () => ({ items: [] })),
    },
    agents: {
      get: ag.get ?? (async () => sdkAgent),
      create: ag.create ?? (async () => sdkAgent),
      update: ag.update ?? (async () => sdkAgent),
      delete: ag.delete ?? (async () => undefined),
    },
  } as SdkClient;
}

describe('createScheduledSessionsAdapter', () => {
  describe('listScheduledSessions', () => {
    it('returns scheduled sessions with resolved agent templates', async () => {
      const adapter = createScheduledSessionsAdapter(makeFakeClient());

      const result = await adapter.listScheduledSessions('proj-1');

      expect(result).toHaveLength(1);
      expect(result[0].name).toBe('daily-build');
      expect(result[0].schedule).toBe('0 9 * * *');
      expect(result[0].suspend).toBe(false);
      expect(result[0].sessionTemplate.initialPrompt).toBe('Run the build');
      expect(result[0].sessionTemplate.llmSettings?.model).toBe('claude-sonnet-4-5');
    });

    it('handles missing agents gracefully via allSettled', async () => {
      const adapter = createScheduledSessionsAdapter(makeFakeClient({
        agents: {
          get: async () => { throw new Error('agent deleted'); },
        },
      }));

      const result = await adapter.listScheduledSessions('proj-1');

      expect(result).toHaveLength(1);
      expect(result[0].sessionTemplate.initialPrompt).toBe('Run tests');
    });

    it('deduplicates agent lookups for shared agent_ids', async () => {
      const agentGetCalls: string[] = [];
      const twoSessions: SdkScheduledSession[] = [
        { ...sdkScheduledSession, name: 'ss-1', agent_id: 'agent-1' },
        { ...sdkScheduledSession, name: 'ss-2', agent_id: 'agent-1' },
      ];
      const adapter = createScheduledSessionsAdapter(makeFakeClient({
        scheduledSessions: {
          list: async () => ({ kind: 'ScheduledSessionList', items: twoSessions, total: 2, page: 1, size: 1000 }),
        },
        agents: {
          get: async (id) => { agentGetCalls.push(id); return sdkAgent; },
        },
      }));

      await adapter.listScheduledSessions('proj-1');

      expect(agentGetCalls).toEqual(['agent-1']);
    });

    it('skips agent resolution when no agent_ids present', async () => {
      let agentGetCalled = false;
      const adapter = createScheduledSessionsAdapter(makeFakeClient({
        scheduledSessions: {
          list: async () => ({
            kind: 'ScheduledSessionList',
            items: [{ ...sdkScheduledSession, agent_id: '' }],
            total: 1, page: 1, size: 1000,
          }),
        },
        agents: {
          get: async () => { agentGetCalled = true; return sdkAgent; },
        },
      }));

      await adapter.listScheduledSessions('proj-1');

      expect(agentGetCalled).toBe(false);
    });
  });

  describe('getScheduledSession', () => {
    it('returns session with resolved agent', async () => {
      const adapter = createScheduledSessionsAdapter(makeFakeClient());

      const result = await adapter.getScheduledSession('proj-1', 'daily-build');

      expect(result.name).toBe('daily-build');
      expect(result.sessionTemplate.initialPrompt).toBe('Run the build');
    });

    it('handles deleted agent gracefully', async () => {
      const adapter = createScheduledSessionsAdapter(makeFakeClient({
        agents: {
          get: async () => { throw new Error('not found'); },
        },
      }));

      const result = await adapter.getScheduledSession('proj-1', 'daily-build');

      expect(result.name).toBe('daily-build');
      expect(result.sessionTemplate.initialPrompt).toBe('Run tests');
    });

    it('skips agent lookup when agent_id is empty', async () => {
      let agentGetCalled = false;
      const adapter = createScheduledSessionsAdapter(makeFakeClient({
        scheduledSessions: {
          get: async () => ({ ...sdkScheduledSession, agent_id: '' }),
        },
        agents: {
          get: async () => { agentGetCalled = true; return sdkAgent; },
        },
      }));

      await adapter.getScheduledSession('proj-1', 'daily-build');

      expect(agentGetCalled).toBe(false);
    });
  });

  describe('createScheduledSession', () => {
    it('creates agent then scheduled session', async () => {
      const calls: string[] = [];
      let agentCreateData: unknown;
      let ssCreateData: unknown;
      const adapter = createScheduledSessionsAdapter(makeFakeClient({
        agents: {
          create: async (data) => { calls.push('agent.create'); agentCreateData = data; return sdkAgent; },
        },
        scheduledSessions: {
          create: async (data) => { calls.push('ss.create'); ssCreateData = data; return sdkScheduledSession; },
        },
      }));

      const result = await adapter.createScheduledSession('proj-1', {
        schedule: '0 12 * * 1',
        displayName: 'Weekly Review',
        sessionTemplate: {
          initialPrompt: 'Review code',
          llmSettings: { model: 'claude-sonnet-4-5', temperature: 0.7, maxTokens: 4096 },
        },
      });

      expect(calls).toEqual(['agent.create', 'ss.create']);
      expect(agentCreateData).toEqual(expect.objectContaining({
        name: 'Weekly Review',
        project_id: 'proj-1',
        prompt: 'Review code',
        llm_model: 'claude-sonnet-4-5',
      }));
      expect(ssCreateData).toEqual(expect.objectContaining({
        schedule: '0 12 * * 1',
        agent_id: 'agent-1',
        enabled: true,
      }));
      expect(result.name).toBe('daily-build');
    });
  });

  describe('updateScheduledSession', () => {
    it('patches agent when sessionTemplate provided', async () => {
      let agentPatched = false;
      let agentPatchData: unknown;
      const adapter = createScheduledSessionsAdapter(makeFakeClient({
        agents: {
          get: async () => sdkAgent,
          update: async (_id, data) => { agentPatched = true; agentPatchData = data; return sdkAgent; },
        },
      }));

      await adapter.updateScheduledSession('proj-1', 'daily-build', {
        sessionTemplate: { initialPrompt: 'Updated prompt', llmSettings: { model: 'claude-opus-4-6' } },
      });

      expect(agentPatched).toBe(true);
      expect(agentPatchData).toEqual(expect.objectContaining({
        prompt: 'Updated prompt',
        llm_model: 'claude-opus-4-6',
      }));
    });

    it('patches scheduled session when schedule provided', async () => {
      let ssUpdateData: unknown;
      const adapter = createScheduledSessionsAdapter(makeFakeClient({
        scheduledSessions: {
          update: async (_id, data) => { ssUpdateData = data; return sdkScheduledSession; },
        },
      }));

      await adapter.updateScheduledSession('proj-1', 'daily-build', {
        schedule: '0 10 * * *',
        suspend: true,
      });

      expect(ssUpdateData).toEqual(expect.objectContaining({
        schedule: '0 10 * * *',
        enabled: false,
      }));
    });

    it('skips scheduled session update when no relevant fields change', async () => {
      let ssUpdateCalled = false;
      const adapter = createScheduledSessionsAdapter(makeFakeClient({
        scheduledSessions: {
          update: async () => { ssUpdateCalled = true; return sdkScheduledSession; },
        },
        agents: {
          update: async () => sdkAgent,
        },
      }));

      await adapter.updateScheduledSession('proj-1', 'daily-build', {
        sessionTemplate: { initialPrompt: 'Only agent change' },
      });

      expect(ssUpdateCalled).toBe(false);
    });
  });

  describe('deleteScheduledSession', () => {
    it('deletes scheduled session and its agent', async () => {
      const deletedIds: string[] = [];
      const adapter = createScheduledSessionsAdapter(makeFakeClient({
        scheduledSessions: {
          delete: async (id) => { deletedIds.push(`ss:${id}`); },
        },
        agents: {
          delete: async (id) => { deletedIds.push(`agent:${id}`); },
        },
      }));

      await adapter.deleteScheduledSession('proj-1', 'daily-build');

      expect(deletedIds).toContain('ss:daily-build');
      expect(deletedIds).toContain('agent:agent-1');
    });

    it('succeeds even if agent deletion fails', async () => {
      const adapter = createScheduledSessionsAdapter(makeFakeClient({
        agents: {
          delete: async () => { throw new Error('already gone'); },
        },
      }));

      await expect(adapter.deleteScheduledSession('proj-1', 'daily-build')).resolves.toBeUndefined();
    });
  });

  describe('suspendScheduledSession', () => {
    it('suspends and returns updated session', async () => {
      let suspendedId: string | undefined;
      const adapter = createScheduledSessionsAdapter(makeFakeClient({
        scheduledSessions: {
          suspend: async (id) => { suspendedId = id; return { ...sdkScheduledSession, enabled: false }; },
        },
      }));

      const result = await adapter.suspendScheduledSession('proj-1', 'daily-build');

      expect(suspendedId).toBe('daily-build');
      expect(result.suspend).toBe(true);
    });
  });

  describe('resumeScheduledSession', () => {
    it('resumes and returns updated session', async () => {
      let resumedId: string | undefined;
      const adapter = createScheduledSessionsAdapter(makeFakeClient({
        scheduledSessions: {
          resume: async (id) => { resumedId = id; return { ...sdkScheduledSession, enabled: true }; },
        },
      }));

      const result = await adapter.resumeScheduledSession('proj-1', 'daily-build');

      expect(resumedId).toBe('daily-build');
      expect(result.suspend).toBe(false);
    });
  });

  describe('triggerScheduledSession', () => {
    it('returns triggered run name and namespace', async () => {
      const adapter = createScheduledSessionsAdapter(makeFakeClient({
        scheduledSessions: {
          trigger: async () => ({ name: 'manual-run-123', namespace: 'proj-ns' }),
        },
      }));

      const result = await adapter.triggerScheduledSession('proj-1', 'daily-build');

      expect(result.name).toBe('manual-run-123');
      expect(result.namespace).toBe('proj-ns');
    });
  });

  describe('listScheduledSessionRuns', () => {
    it('returns runs from SDK response', async () => {
      const fakeRuns = [{ metadata: { name: 'run-1' }, status: { phase: 'Running' } }];
      const adapter = createScheduledSessionsAdapter(makeFakeClient({
        scheduledSessions: {
          runs: async () => ({ items: fakeRuns }),
        },
      }));

      const result = await adapter.listScheduledSessionRuns('proj-1', 'daily-build');

      expect(result).toHaveLength(1);
    });

    it('returns empty array when no items in response', async () => {
      const adapter = createScheduledSessionsAdapter(makeFakeClient({
        scheduledSessions: {
          runs: async () => ({}),
        },
      }));

      const result = await adapter.listScheduledSessionRuns('proj-1', 'daily-build');

      expect(result).toEqual([]);
    });
  });

  describe('error handling', () => {
    it('wraps SDK errors as ApiClientError', async () => {
      const { AmbientAPIError } = await import('@/lib/sdk');
      const adapter = createScheduledSessionsAdapter(makeFakeClient({
        scheduledSessions: {
          get: async () => {
            throw new AmbientAPIError({
              id: '', kind: 'Error', href: '',
              code: 'NOT_FOUND', reason: 'Not found',
              operation_id: 'op-1', status_code: 404,
            });
          },
        },
      }));

      await expect(adapter.getScheduledSession('proj-1', 'missing')).rejects.toThrow(ApiClientError);
    });
  });
});
