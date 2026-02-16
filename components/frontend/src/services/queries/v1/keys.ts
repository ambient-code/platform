import type { ListOptions } from '@ambient-platform/sdk';

const v1 = 'v1' as const;

export const agentKeys = {
  all: [v1, 'agents'] as const,
  lists: () => [...agentKeys.all, 'list'] as const,
  list: (opts?: ListOptions) => [...agentKeys.lists(), opts ?? {}] as const,
  details: () => [...agentKeys.all, 'detail'] as const,
  detail: (id: string) => [...agentKeys.details(), id] as const,
};

export const sessionKeys = {
  all: [v1, 'sessions'] as const,
  lists: () => [...sessionKeys.all, 'list'] as const,
  list: (opts?: ListOptions) => [...sessionKeys.lists(), opts ?? {}] as const,
  details: () => [...sessionKeys.all, 'detail'] as const,
  detail: (id: string) => [...sessionKeys.details(), id] as const,
};

export const skillKeys = {
  all: [v1, 'skills'] as const,
  lists: () => [...skillKeys.all, 'list'] as const,
  list: (opts?: ListOptions) => [...skillKeys.lists(), opts ?? {}] as const,
  details: () => [...skillKeys.all, 'detail'] as const,
  detail: (id: string) => [...skillKeys.details(), id] as const,
};

export const taskKeys = {
  all: [v1, 'tasks'] as const,
  lists: () => [...taskKeys.all, 'list'] as const,
  list: (opts?: ListOptions) => [...taskKeys.lists(), opts ?? {}] as const,
  details: () => [...taskKeys.all, 'detail'] as const,
  detail: (id: string) => [...taskKeys.details(), id] as const,
};

export const userKeys = {
  all: [v1, 'users'] as const,
  lists: () => [...userKeys.all, 'list'] as const,
  list: (opts?: ListOptions) => [...userKeys.lists(), opts ?? {}] as const,
  details: () => [...userKeys.all, 'detail'] as const,
  detail: (id: string) => [...userKeys.details(), id] as const,
};

export const workflowKeys = {
  all: [v1, 'workflows'] as const,
  lists: () => [...workflowKeys.all, 'list'] as const,
  list: (opts?: ListOptions) => [...workflowKeys.lists(), opts ?? {}] as const,
  details: () => [...workflowKeys.all, 'detail'] as const,
  detail: (id: string) => [...workflowKeys.details(), id] as const,
};

export const workflowSkillKeys = {
  all: [v1, 'workflowSkills'] as const,
  lists: () => [...workflowSkillKeys.all, 'list'] as const,
  list: (opts?: ListOptions) => [...workflowSkillKeys.lists(), opts ?? {}] as const,
  details: () => [...workflowSkillKeys.all, 'detail'] as const,
  detail: (id: string) => [...workflowSkillKeys.details(), id] as const,
};

export const workflowTaskKeys = {
  all: [v1, 'workflowTasks'] as const,
  lists: () => [...workflowTaskKeys.all, 'list'] as const,
  list: (opts?: ListOptions) => [...workflowTaskKeys.lists(), opts ?? {}] as const,
  details: () => [...workflowTaskKeys.all, 'detail'] as const,
  detail: (id: string) => [...workflowTaskKeys.details(), id] as const,
};

export const projectKeys = {
  all: [v1, 'projects'] as const,
  lists: () => [...projectKeys.all, 'list'] as const,
  list: (opts?: ListOptions) => [...projectKeys.lists(), opts ?? {}] as const,
  details: () => [...projectKeys.all, 'detail'] as const,
  detail: (id: string) => [...projectKeys.details(), id] as const,
};

export const projectSettingsKeys = {
  all: [v1, 'projectSettings'] as const,
  lists: () => [...projectSettingsKeys.all, 'list'] as const,
  list: (opts?: ListOptions) => [...projectSettingsKeys.lists(), opts ?? {}] as const,
  details: () => [...projectSettingsKeys.all, 'detail'] as const,
  detail: (id: string) => [...projectSettingsKeys.details(), id] as const,
};

export const permissionKeys = {
  all: [v1, 'permissions'] as const,
  lists: () => [...permissionKeys.all, 'list'] as const,
  list: (opts?: ListOptions) => [...permissionKeys.lists(), opts ?? {}] as const,
  details: () => [...permissionKeys.all, 'detail'] as const,
  detail: (id: string) => [...permissionKeys.details(), id] as const,
};

export const repositoryRefKeys = {
  all: [v1, 'repositoryRefs'] as const,
  lists: () => [...repositoryRefKeys.all, 'list'] as const,
  list: (opts?: ListOptions) => [...repositoryRefKeys.lists(), opts ?? {}] as const,
  details: () => [...repositoryRefKeys.all, 'detail'] as const,
  detail: (id: string) => [...repositoryRefKeys.details(), id] as const,
};

export const projectKeyKeys = {
  all: [v1, 'projectKeys'] as const,
  lists: () => [...projectKeyKeys.all, 'list'] as const,
  list: (opts?: ListOptions) => [...projectKeyKeys.lists(), opts ?? {}] as const,
  details: () => [...projectKeyKeys.all, 'detail'] as const,
  detail: (id: string) => [...projectKeyKeys.details(), id] as const,
};
