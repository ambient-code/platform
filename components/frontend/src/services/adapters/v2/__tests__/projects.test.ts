import { describe, it, expect } from 'vitest';
import { createProjectsAdapter } from '../projects';
import { ApiClientError } from '@/types/api/common';
import type { SdkClient } from '../client';
import type { Project as SdkProject, ProjectList } from '@/lib/sdk';

const sdkProject: SdkProject = {
  id: 'proj-uuid',
  kind: 'Project',
  href: '/api/ambient/v1/projects/proj-uuid',
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-02T00:00:00Z',
  name: 'my-project',
  display_name: 'My Project',
  description: 'A test project',
  labels: '{"env":"dev"}',
  annotations: '{"team":"platform"}',
  status: 'active',
  prompt: '',
};

function makeFakeClient(overrides?: {
  list?: (opts?: unknown) => Promise<ProjectList>;
  get?: (id: string) => Promise<SdkProject>;
  create?: (data: unknown) => Promise<SdkProject>;
  update?: (id: string, data: unknown) => Promise<SdkProject>;
  delete?: (id: string) => Promise<void>;
}): SdkClient {
  return {
    projects: {
      list: overrides?.list ?? (async () => ({ kind: 'ProjectList', items: [sdkProject], total: 1, page: 1, size: 20 })),
      get: overrides?.get ?? (async () => sdkProject),
      create: overrides?.create ?? (async () => sdkProject),
      update: overrides?.update ?? (async () => sdkProject),
      delete: overrides?.delete ?? (async () => undefined),
    },
    sessions: {} as SdkClient['sessions'],
    scheduledSessions: {} as SdkClient['scheduledSessions'],
    agents: {} as SdkClient['agents'],
  } as SdkClient;
}

describe('createProjectsAdapter', () => {
  describe('listProjects', () => {
    it('transforms SDK list to PaginatedResult', async () => {
      const adapter = createProjectsAdapter(makeFakeClient());

      const result = await adapter.listProjects();

      expect(result.items).toHaveLength(1);
      expect(result.items[0].name).toBe('my-project');
      expect(result.items[0].displayName).toBe('My Project');
      expect(result.totalCount).toBe(1);
      expect(result.hasMore).toBe(false);
      expect(result.nextPage).toBeUndefined();
    });

    it('parses JSON string labels and annotations', async () => {
      const adapter = createProjectsAdapter(makeFakeClient());

      const result = await adapter.listProjects();

      expect(result.items[0].labels).toEqual({ env: 'dev' });
      expect(result.items[0].annotations).toEqual({ team: 'platform' });
    });

    it('provides nextPage when hasMore is true', async () => {
      let callCount = 0;
      const adapter = createProjectsAdapter(makeFakeClient({
        list: async () => {
          callCount++;
          return callCount === 1
            ? { kind: 'ProjectList', items: [sdkProject], total: 50, page: 1, size: 20 }
            : { kind: 'ProjectList', items: [sdkProject], total: 50, page: 2, size: 20 };
        },
      }));

      const result = await adapter.listProjects();

      expect(result.hasMore).toBe(true);
      expect(result.nextPage).toBeDefined();

      const page2 = await result.nextPage!();
      expect(page2.items).toHaveLength(1);
    });

    it('passes pagination params to SDK', async () => {
      let receivedOpts: unknown;
      const adapter = createProjectsAdapter(makeFakeClient({
        list: async (opts) => {
          receivedOpts = opts;
          return { kind: 'ProjectList', items: [], total: 0, page: 2, size: 10 };
        },
      }));

      await adapter.listProjects({ offset: 10, limit: 10 });

      expect(receivedOpts).toEqual({ page: 2, size: 10, search: undefined });
    });

    it('passes search param to SDK', async () => {
      let receivedOpts: unknown;
      const adapter = createProjectsAdapter(makeFakeClient({
        list: async (opts) => {
          receivedOpts = opts;
          return { kind: 'ProjectList', items: [], total: 0, page: 1, size: 20 };
        },
      }));

      await adapter.listProjects({ search: 'test' });

      expect(receivedOpts).toEqual(expect.objectContaining({ search: 'test' }));
    });
  });

  describe('getProject', () => {
    it('transforms SDK project to canonical type', async () => {
      const adapter = createProjectsAdapter(makeFakeClient());

      const result = await adapter.getProject('my-project');

      expect(result.name).toBe('my-project');
      expect(result.displayName).toBe('My Project');
      expect(result.description).toBe('A test project');
      expect(result.creationTimestamp).toBe('2026-01-01T00:00:00Z');
      expect(result.status).toBe('active');
      expect(result.isOpenShift).toBe(false);
      expect(result.uid).toBe('proj-uuid');
    });

    it('handles missing optional fields', async () => {
      const adapter = createProjectsAdapter(makeFakeClient({
        get: async () => ({
          ...sdkProject,
          display_name: '',
          description: '',
          labels: '',
          annotations: '',
          created_at: '',
          status: '',
        }),
      }));

      const result = await adapter.getProject('my-project');

      expect(result.displayName).toBe('');
      expect(result.description).toBeUndefined();
      expect(result.labels).toEqual({});
      expect(result.annotations).toEqual({});
      expect(result.status).toBe('active');
    });

    it('handles labels/annotations as pre-parsed objects', async () => {
      const adapter = createProjectsAdapter(makeFakeClient({
        get: async () => ({
          ...sdkProject,
          labels: { env: 'prod' } as unknown as string,
          annotations: { team: 'infra' } as unknown as string,
        }),
      }));

      const result = await adapter.getProject('my-project');

      expect(result.labels).toEqual({ env: 'prod' });
      expect(result.annotations).toEqual({ team: 'infra' });
    });
  });

  describe('createProject', () => {
    it('sends correct create request to SDK', async () => {
      let receivedData: unknown;
      const adapter = createProjectsAdapter(makeFakeClient({
        create: async (data) => {
          receivedData = data;
          return sdkProject;
        },
      }));

      await adapter.createProject({
        name: 'new-proj',
        displayName: 'New Project',
        description: 'desc',
        labels: { env: 'staging' },
      });

      expect(receivedData).toEqual({
        name: 'new-proj',
        display_name: 'New Project',
        description: 'desc',
        labels: '{"env":"staging"}',
      });
    });

    it('omits labels when not provided', async () => {
      let receivedData: unknown;
      const adapter = createProjectsAdapter(makeFakeClient({
        create: async (data) => {
          receivedData = data;
          return sdkProject;
        },
      }));

      await adapter.createProject({ name: 'new-proj' });

      expect(receivedData).toEqual(expect.objectContaining({ labels: undefined }));
    });
  });

  describe('updateProject', () => {
    it('sends correct update request to SDK', async () => {
      let receivedName: string | undefined;
      let receivedData: unknown;
      const adapter = createProjectsAdapter(makeFakeClient({
        update: async (name, data) => {
          receivedName = name;
          receivedData = data;
          return sdkProject;
        },
      }));

      await adapter.updateProject('my-project', {
        displayName: 'Updated',
        labels: { env: 'prod' },
        annotations: { owner: 'team-a' },
      });

      expect(receivedName).toBe('my-project');
      expect(receivedData).toEqual({
        display_name: 'Updated',
        description: undefined,
        labels: '{"env":"prod"}',
        annotations: '{"owner":"team-a"}',
      });
    });
  });

  describe('deleteProject', () => {
    it('returns the project name after deletion', async () => {
      let deletedName: string | undefined;
      const adapter = createProjectsAdapter(makeFakeClient({
        delete: async (name) => { deletedName = name; },
      }));

      const result = await adapter.deleteProject('my-project');

      expect(result).toBe('my-project');
      expect(deletedName).toBe('my-project');
    });
  });

  describe('NOT_IMPLEMENTED methods', () => {
    const adapter = createProjectsAdapter(makeFakeClient());

    it('getProjectIntegrationStatus throws NOT_IMPLEMENTED', async () => {
      await expect(adapter.getProjectIntegrationStatus('proj')).rejects.toThrow(ApiClientError);
      await expect(adapter.getProjectIntegrationStatus('proj')).rejects.toThrow('Not implemented');
    });

    it('getProjectMcpServers throws NOT_IMPLEMENTED', async () => {
      await expect(adapter.getProjectMcpServers('proj')).rejects.toThrow(ApiClientError);
    });

    it('updateProjectMcpServers throws NOT_IMPLEMENTED', async () => {
      await expect(adapter.updateProjectMcpServers('proj', {})).rejects.toThrow(ApiClientError);
    });
  });

  describe('error handling', () => {
    it('wraps SDK errors as ApiClientError', async () => {
      const { AmbientAPIError } = await import('@/lib/sdk');
      const adapter = createProjectsAdapter(makeFakeClient({
        get: async () => {
          throw new AmbientAPIError({
            id: '', kind: 'Error', href: '',
            code: 'NOT_FOUND', reason: 'Project not found',
            operation_id: 'op-1', status_code: 404,
          });
        },
      }));

      await expect(adapter.getProject('missing')).rejects.toThrow(ApiClientError);
      try {
        await adapter.getProject('missing');
      } catch (e) {
        const err = e as ApiClientError;
        expect(err.message).toBe('Project not found');
        expect(err.code).toBe('NOT_FOUND');
      }
    });
  });
});
