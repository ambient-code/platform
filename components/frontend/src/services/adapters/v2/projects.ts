import type { Project as SdkProject } from '@/lib/sdk';
import type { ProjectsPort } from '../../ports/projects';
import type { Project } from '@/types/api/projects';
import type { PaginationParams } from '@/types/api/common';
import { ApiClientError } from '@/types/api/common';
import { getClient } from './client';
import { toListOptions, fromSdkList } from './pagination';
import { parseJsonField } from './json';
import { wrapSdkError } from './errors';

function toProject(s: SdkProject): Project {
  return {
    name: s.name,
    displayName: s.display_name || '',
    description: s.description || undefined,
    labels: parseJsonField<Record<string, string>>(s.labels, {}),
    annotations: parseJsonField<Record<string, string>>(s.annotations, {}),
    creationTimestamp: s.created_at || '',
    status: (s.status as Project['status']) || 'active',
    isOpenShift: false,
    uid: s.id,
  };
}

export function createProjectsAdapter(): ProjectsPort {
  return {
    async listProjects(params?: PaginationParams) {
      try {
        const client = getClient();
        const opts = toListOptions(params);
        const list = await client.projects.list(opts);
        return fromSdkList(list, toProject, (p) => this.listProjects(p));
      } catch (err) {
        wrapSdkError(err);
      }
    },

    async getProject(name: string) {
      try {
        const client = getClient();
        const sdk = await client.projects.get(name);
        return toProject(sdk);
      } catch (err) {
        wrapSdkError(err);
      }
    },

    async createProject(data) {
      try {
        const client = getClient();
        const sdk = await client.projects.create({
          name: data.name,
          display_name: data.displayName,
          description: data.description,
          labels: data.labels ? JSON.stringify(data.labels) : undefined,
        });
        return toProject(sdk);
      } catch (err) {
        wrapSdkError(err);
      }
    },

    async updateProject(name: string, data) {
      try {
        const client = getClient();
        const sdk = await client.projects.update(name, {
          display_name: data.displayName,
          description: data.description,
          labels: data.labels ? JSON.stringify(data.labels) : undefined,
          annotations: data.annotations ? JSON.stringify(data.annotations) : undefined,
        });
        return toProject(sdk);
      } catch (err) {
        wrapSdkError(err);
      }
    },

    async deleteProject(name: string) {
      try {
        const client = getClient();
        await client.projects.delete(name);
        return name;
      } catch (err) {
        wrapSdkError(err);
      }
    },

    async getProjectIntegrationStatus() {
      throw new ApiClientError('Not implemented in v2 adapter', 'NOT_IMPLEMENTED');
    },

    async getProjectMcpServers() {
      throw new ApiClientError('Not implemented in v2 adapter', 'NOT_IMPLEMENTED');
    },

    async updateProjectMcpServers() {
      throw new ApiClientError('Not implemented in v2 adapter', 'NOT_IMPLEMENTED');
    },
  };
}

export const projectsAdapter = createProjectsAdapter();
