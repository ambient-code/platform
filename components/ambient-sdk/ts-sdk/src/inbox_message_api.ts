import type { AmbientClientConfig, ListOptions, RequestOptions, ListMeta } from './base';
import { ambientFetch, buildQueryString } from './base';
import type { InboxMessage, InboxMessageCreateRequest } from './inbox_message';

export type InboxMessageList = ListMeta & { items: InboxMessage[] };

export class InboxMessageAPI {
  constructor(private readonly config: AmbientClientConfig) {}

  async send(projectId: string, agentId: string, data: InboxMessageCreateRequest, opts?: RequestOptions): Promise<InboxMessage> {
    return ambientFetch<InboxMessage>(this.config, 'POST', `/projects/${encodeURIComponent(projectId)}/agents/${encodeURIComponent(agentId)}/inbox`, data, opts);
  }

  async listByAgent(projectId: string, agentId: string, listOpts?: ListOptions, opts?: RequestOptions): Promise<InboxMessageList> {
    const qs = buildQueryString(listOpts);
    return ambientFetch<InboxMessageList>(this.config, 'GET', `/projects/${encodeURIComponent(projectId)}/agents/${encodeURIComponent(agentId)}/inbox${qs}`, undefined, opts);
  }

  async markRead(projectId: string, agentId: string, msgId: string, opts?: RequestOptions): Promise<void> {
    return ambientFetch<void>(this.config, 'PATCH', `/projects/${encodeURIComponent(projectId)}/agents/${encodeURIComponent(agentId)}/inbox/${encodeURIComponent(msgId)}`, { read: true }, opts);
  }

  async deleteMessage(projectId: string, agentId: string, msgId: string, opts?: RequestOptions): Promise<void> {
    return ambientFetch<void>(this.config, 'DELETE', `/projects/${encodeURIComponent(projectId)}/agents/${encodeURIComponent(agentId)}/inbox/${encodeURIComponent(msgId)}`, undefined, opts);
  }
}
