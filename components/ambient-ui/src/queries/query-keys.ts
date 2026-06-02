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
  messages: {
    all: ['messages'] as const,
    lists: () => [...queryKeys.messages.all, 'list'] as const,
    list: (sessionId: string) =>
      [...queryKeys.messages.lists(), sessionId] as const,
  },
} as const
