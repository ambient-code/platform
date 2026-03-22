import type { AmbientClientConfig, ListOptions, RequestOptions, ListMeta } from './base';
import { ambientFetch, buildQueryString } from './base';
import type { ProjectAgent, ProjectAgentCreateRequest, ProjectAgentPatchRequest } from './project_agent';
import type { Session } from './session';

export type ProjectAgentList = ListMeta & { items: ProjectAgent[] };
export type SessionList = ListMeta & { items: Session[] };

export type IgniteRequest = {
  prompt?: string;
};

export type IgniteResponse = {
  session?: Session;
  ignition_prompt?: string;
};

export class ProjectAgentAPI {
  constructor(private readonly config: AmbientClientConfig) {}

  async listByProject(projectId: string, listOpts?: ListOptions, opts?: RequestOptions): Promise<ProjectAgentList> {
    const qs = buildQueryString(listOpts);
    return ambientFetch<ProjectAgentList>(this.config, 'GET', `/projects/${encodeURIComponent(projectId)}/agents${qs}`, undefined, opts);
  }

  async getByProject(projectId: string, agentId: string, opts?: RequestOptions): Promise<ProjectAgent> {
    return ambientFetch<ProjectAgent>(this.config, 'GET', `/projects/${encodeURIComponent(projectId)}/agents/${encodeURIComponent(agentId)}`, undefined, opts);
  }

  async createInProject(projectId: string, data: ProjectAgentCreateRequest, opts?: RequestOptions): Promise<ProjectAgent> {
    return ambientFetch<ProjectAgent>(this.config, 'POST', `/projects/${encodeURIComponent(projectId)}/agents`, data, opts);
  }

  async updateInProject(projectId: string, agentId: string, patch: ProjectAgentPatchRequest, opts?: RequestOptions): Promise<ProjectAgent> {
    return ambientFetch<ProjectAgent>(this.config, 'PATCH', `/projects/${encodeURIComponent(projectId)}/agents/${encodeURIComponent(agentId)}`, patch, opts);
  }

  async deleteInProject(projectId: string, agentId: string, opts?: RequestOptions): Promise<void> {
    return ambientFetch<void>(this.config, 'DELETE', `/projects/${encodeURIComponent(projectId)}/agents/${encodeURIComponent(agentId)}`, undefined, opts);
  }

  async ignite(projectId: string, agentId: string, prompt?: string, opts?: RequestOptions): Promise<IgniteResponse> {
    return ambientFetch<IgniteResponse>(this.config, 'POST', `/projects/${encodeURIComponent(projectId)}/agents/${encodeURIComponent(agentId)}/ignite`, { prompt }, opts);
  }

  async getIgnition(projectId: string, agentId: string, opts?: RequestOptions): Promise<IgniteResponse> {
    return ambientFetch<IgniteResponse>(this.config, 'GET', `/projects/${encodeURIComponent(projectId)}/agents/${encodeURIComponent(agentId)}/ignition`, undefined, opts);
  }

  async sessions(projectId: string, agentId: string, listOpts?: ListOptions, opts?: RequestOptions): Promise<SessionList> {
    const qs = buildQueryString(listOpts);
    return ambientFetch<SessionList>(this.config, 'GET', `/projects/${encodeURIComponent(projectId)}/agents/${encodeURIComponent(agentId)}/sessions${qs}`, undefined, opts);
  }
}
