import { describe, it, expect, vi } from 'vitest';
import { createSessionsAdapter } from '../sessions';
import { ApiClientError } from '@/types/api/common';
import type { SdkClient } from '../client';
import type { Session as SdkSession } from '@/lib/sdk';

const sdkSession: SdkSession = {
  id: 'sess-uuid',
  kind: 'Session',
  href: '/api/ambient/v1/sessions/sess-uuid',
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-02T00:00:00Z',
  name: 'my-session',
  kube_cr_name: 'session-cr',
  kube_namespace: 'ns-1',
  kube_cr_uid: 'cr-uid-1',
  labels: '{"env":"dev"}',
  annotations: '{"team":"platform"}',
  repos: '[{"url":"https://github.com/org/repo"}]',
  environment_variables: '{"API_KEY":"secret"}',
  conditions: '[]',
  reconciled_repos: '[]',
  reconciled_workflow: '',
  resource_overrides: '',
  llm_model: 'claude-sonnet-4-5',
  llm_temperature: 0.7,
  llm_max_tokens: 4096,
  prompt: 'Run tests',
  timeout: 3600,
  phase: 'Running',
  start_time: '2026-01-01T00:01:00Z',
  completion_time: '',
  sdk_session_id: 'sdk-sess-1',
  sdk_restart_count: 0,
  workflow_id: 'wf-1',
  project_id: 'proj-1',
  parent_session_id: '',
  agent_id: '',
  assigned_user_id: '',
  bot_account_name: '',
  created_by_user_id: '',
  triggered_by_user_id: '',
  repo_url: '',
};

function makeFakeClient(overrides?: Partial<{
  list: (opts?: unknown) => Promise<{ items: SdkSession[]; total: number; page: number; size: number }>;
  get: (id: string) => Promise<SdkSession>;
  create: (data: unknown) => Promise<SdkSession>;
  stop: (id: string) => Promise<SdkSession>;
  start: (id: string) => Promise<SdkSession>;
  delete: (id: string) => Promise<void>;
  update: (id: string, data: unknown) => Promise<SdkSession>;
}>): SdkClient {
  return {
    projects: {} as SdkClient['projects'],
    sessions: {
      list: overrides?.list ?? (async () => ({ kind: 'SessionList', items: [sdkSession], total: 1, page: 1, size: 20 })),
      get: overrides?.get ?? (async () => sdkSession),
      create: overrides?.create ?? (async () => sdkSession),
      stop: overrides?.stop ?? (async () => sdkSession),
      start: overrides?.start ?? (async () => sdkSession),
      delete: overrides?.delete ?? (async () => undefined),
      update: overrides?.update ?? (async () => sdkSession),
    },
    scheduledSessions: {} as SdkClient['scheduledSessions'],
    agents: {} as SdkClient['agents'],
  } as SdkClient;
}

describe('createSessionsAdapter', () => {
  describe('listSessions', () => {
    it('transforms SDK list to PaginatedResult', async () => {
      const adapter = createSessionsAdapter(makeFakeClient());

      const result = await adapter.listSessions('proj-1');

      expect(result.items).toHaveLength(1);
      expect(result.totalCount).toBe(1);
      expect(result.hasMore).toBe(false);
    });

    it('maps nested metadata/spec/status structure', async () => {
      const adapter = createSessionsAdapter(makeFakeClient());

      const result = await adapter.listSessions('proj-1');
      const session = result.items[0];

      expect(session.metadata.name).toBe('session-cr');
      expect(session.metadata.namespace).toBe('ns-1');
      expect(session.metadata.uid).toBe('cr-uid-1');
      expect(session.metadata.creationTimestamp).toBe('2026-01-01T00:00:00Z');
      expect(session.metadata.labels).toEqual({ env: 'dev' });
      expect(session.metadata.annotations).toEqual({ team: 'platform' });

      expect(session.spec.initialPrompt).toBe('Run tests');
      expect(session.spec.llmSettings.model).toBe('claude-sonnet-4-5');
      expect(session.spec.llmSettings.temperature).toBe(0.7);
      expect(session.spec.llmSettings.maxTokens).toBe(4096);
      expect(session.spec.timeout).toBe(3600);
      expect(session.spec.displayName).toBe('my-session');
      expect(session.spec.repos).toEqual([{ url: 'https://github.com/org/repo' }]);
      expect(session.spec.environmentVariables).toEqual({ API_KEY: 'secret' });
      expect(session.spec.activeWorkflow).toEqual({ gitUrl: 'wf-1', branch: 'main' });

      expect(session.status?.phase).toBe('Running');
      expect(session.status?.startTime).toBe('2026-01-01T00:01:00Z');
      expect(session.status?.sdkSessionId).toBe('sdk-sess-1');
    });

    it('handles missing optional fields gracefully', async () => {
      const sparseSession: SdkSession = {
        ...sdkSession,
        kube_cr_name: '',
        kube_namespace: '',
        kube_cr_uid: '',
        labels: '',
        annotations: '',
        repos: '',
        environment_variables: '',
        prompt: '',
        workflow_id: '',
        start_time: '',
        completion_time: '',
        sdk_session_id: '',
        phase: '',
      };
      const adapter = createSessionsAdapter(makeFakeClient({
        list: async () => ({ kind: 'SessionList', items: [sparseSession], total: 1, page: 1, size: 20 }),
      }));

      const result = await adapter.listSessions('proj-1');
      const session = result.items[0];

      expect(session.metadata.name).toBe('my-session');
      expect(session.metadata.uid).toBe('sess-uuid');
      expect(session.metadata.labels).toBeUndefined();
      expect(session.metadata.annotations).toBeUndefined();
      expect(session.spec.initialPrompt).toBeUndefined();
      expect(session.spec.repos).toBeUndefined();
      expect(session.spec.environmentVariables).toBeUndefined();
      expect(session.spec.activeWorkflow).toBeUndefined();
      expect(session.status?.phase).toBe('Pending');
      expect(session.status?.startTime).toBeUndefined();
    });

    it('provides nextPage when hasMore is true', async () => {
      let callCount = 0;
      const adapter = createSessionsAdapter(makeFakeClient({
        list: async () => {
          callCount++;
          return callCount === 1
            ? { kind: 'SessionList', items: [sdkSession], total: 50, page: 1, size: 20 }
            : { kind: 'SessionList', items: [sdkSession], total: 50, page: 2, size: 20 };
        },
      }));

      const result = await adapter.listSessions('proj-1');
      expect(result.hasMore).toBe(true);

      const page2 = await result.nextPage!();
      expect(page2.items).toHaveLength(1);
    });
  });

  describe('getSession', () => {
    it('returns transformed session', async () => {
      let receivedId: string | undefined;
      const adapter = createSessionsAdapter(makeFakeClient({
        get: async (id) => { receivedId = id; return sdkSession; },
      }));

      const result = await adapter.getSession('proj-1', 'my-session');

      expect(receivedId).toBe('my-session');
      expect(result.metadata.name).toBe('session-cr');
      expect(result.spec.llmSettings.model).toBe('claude-sonnet-4-5');
      expect(result.status?.phase).toBe('Running');
    });
  });

  describe('createSession', () => {
    it('sends flattened create request to SDK', async () => {
      let receivedData: unknown;
      const adapter = createSessionsAdapter(makeFakeClient({
        create: async (data) => { receivedData = data; return sdkSession; },
      }));

      await adapter.createSession('proj-1', {
        displayName: 'New Session',
        initialPrompt: 'Hello',
        llmSettings: { model: 'claude-sonnet-4-5', temperature: 0.5, maxTokens: 2048 },
        timeout: 1800,
        labels: { env: 'test' },
        annotations: { note: 'ci' },
        environmentVariables: { KEY: 'val' },
        repos: [{ url: 'https://github.com/org/repo' }],
        activeWorkflow: { gitUrl: 'wf-1', branch: 'main' },
      });

      expect(receivedData).toEqual({
        name: 'New Session',
        prompt: 'Hello',
        llm_model: 'claude-sonnet-4-5',
        llm_temperature: 0.5,
        llm_max_tokens: 2048,
        timeout: 1800,
        labels: '{"env":"test"}',
        annotations: '{"note":"ci"}',
        environment_variables: '{"KEY":"val"}',
        repos: '[{"url":"https://github.com/org/repo"}]',
        workflow_id: 'wf-1',
        parent_session_id: undefined,
      });
    });

    it('generates default name when displayName not provided', async () => {
      let receivedData: Record<string, unknown> = {};
      vi.spyOn(Date, 'now').mockReturnValue(1234567890);
      const adapter = createSessionsAdapter(makeFakeClient({
        create: async (data) => { receivedData = data as Record<string, unknown>; return sdkSession; },
      }));

      await adapter.createSession('proj-1', { initialPrompt: 'test' });

      expect(receivedData.name).toBe('session-1234567890');
      vi.restoreAllMocks();
    });
  });

  describe('stopSession', () => {
    it('calls SDK stop and returns message', async () => {
      let stoppedId: string | undefined;
      const adapter = createSessionsAdapter(makeFakeClient({
        stop: async (id) => { stoppedId = id; return sdkSession; },
      }));

      const result = await adapter.stopSession('proj-1', 'my-session');

      expect(stoppedId).toBe('my-session');
      expect(result).toBe('Session stopped');
    });
  });

  describe('startSession', () => {
    it('calls SDK start and returns message', async () => {
      let startedId: string | undefined;
      const adapter = createSessionsAdapter(makeFakeClient({
        start: async (id) => { startedId = id; return sdkSession; },
      }));

      const result = await adapter.startSession('proj-1', 'my-session');

      expect(startedId).toBe('my-session');
      expect(result).toEqual({ message: 'Session started' });
    });
  });

  describe('deleteSession', () => {
    it('calls SDK delete', async () => {
      let deletedId: string | undefined;
      const adapter = createSessionsAdapter(makeFakeClient({
        delete: async (id) => { deletedId = id; },
      }));

      await adapter.deleteSession('proj-1', 'my-session');

      expect(deletedId).toBe('my-session');
    });
  });

  describe('updateSessionDisplayName', () => {
    it('calls SDK update with name field', async () => {
      let receivedId: string | undefined;
      let receivedPatch: unknown;
      const adapter = createSessionsAdapter(makeFakeClient({
        update: async (id, data) => { receivedId = id; receivedPatch = data; return sdkSession; },
      }));

      const result = await adapter.updateSessionDisplayName('proj-1', 'my-session', 'New Name');

      expect(receivedId).toBe('my-session');
      expect(receivedPatch).toEqual({ name: 'New Name' });
      expect(result.metadata.name).toBe('session-cr');
    });
  });

  describe('switchSessionModel', () => {
    it('calls SDK update with llm_model field', async () => {
      let receivedPatch: unknown;
      const adapter = createSessionsAdapter(makeFakeClient({
        update: async (_id, data) => { receivedPatch = data; return sdkSession; },
      }));

      const result = await adapter.switchSessionModel('proj-1', 'my-session', 'claude-opus-4-6');

      expect(receivedPatch).toEqual({ llm_model: 'claude-opus-4-6' });
      expect(result.metadata.name).toBe('session-cr');
    });
  });

  describe('NOT_IMPLEMENTED methods', () => {
    it('saveToGoogleDrive throws NOT_IMPLEMENTED', async () => {
      const adapter = createSessionsAdapter(makeFakeClient());

      await expect(
        adapter.saveToGoogleDrive('proj-1', 'sess-1', 'content', 'file.md', 'user@example.com'),
      ).rejects.toThrow(ApiClientError);
    });
  });

  describe('error handling', () => {
    it('wraps SDK errors as ApiClientError', async () => {
      const { AmbientAPIError } = await import('@/lib/sdk');
      const adapter = createSessionsAdapter(makeFakeClient({
        get: async () => {
          throw new AmbientAPIError({
            id: '', kind: 'Error', href: '',
            code: 'NOT_FOUND', reason: 'Session not found',
            operation_id: 'op-1', status_code: 404,
          });
        },
      }));

      await expect(adapter.getSession('proj-1', 'missing')).rejects.toThrow(ApiClientError);
    });
  });
});
