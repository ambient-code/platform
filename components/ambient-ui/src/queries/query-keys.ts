import type { ListParams } from '@/domain/types'

export const queryKeys = {
  sessions: {
    all: ['sessions'] as const,
    lists: () => [...queryKeys.sessions.all, 'list'] as const,
    list: (projectId: string, params?: ListParams) =>
      [...queryKeys.sessions.lists(), projectId, params] as const,
    details: () => [...queryKeys.sessions.all, 'detail'] as const,
    detail: (sessionId: string) =>
      [...queryKeys.sessions.details(), sessionId] as const,
  },
  projects: {
    all: ['projects'] as const,
    lists: () => [...queryKeys.projects.all, 'list'] as const,
    list: (params?: ListParams) =>
      [...queryKeys.projects.lists(), params] as const,
    details: () => [...queryKeys.projects.all, 'detail'] as const,
    detail: (projectId: string) =>
      [...queryKeys.projects.details(), projectId] as const,
  },
  agents: {
    all: ['agents'] as const,
    lists: () => [...queryKeys.agents.all, 'list'] as const,
    list: (projectId: string, params?: ListParams) =>
      [...queryKeys.agents.lists(), projectId, params] as const,
    details: () => [...queryKeys.agents.all, 'detail'] as const,
    detail: (projectId: string, agentId: string) =>
      [...queryKeys.agents.details(), projectId, agentId] as const,
    names: (projectId: string) =>
      [...queryKeys.agents.all, 'names', projectId] as const,
  },
  messages: {
    all: ['messages'] as const,
    lists: () => [...queryKeys.messages.all, 'list'] as const,
    list: (sessionId: string) =>
      [...queryKeys.messages.lists(), sessionId] as const,
  },
  credentials: {
    all: ['credentials'] as const,
    lists: () => [...queryKeys.credentials.all, 'list'] as const,
    list: (params?: ListParams) =>
      [...queryKeys.credentials.lists(), params] as const,
    details: () => [...queryKeys.credentials.all, 'detail'] as const,
    detail: (id: string) =>
      [...queryKeys.credentials.details(), id] as const,
  },
  roleBindings: {
    all: ['roleBindings'] as const,
    lists: () => [...queryKeys.roleBindings.all, 'list'] as const,
    list: (params?: ListParams) =>
      [...queryKeys.roleBindings.lists(), params] as const,
  },
} as const
