/**
 * Scheduled Sessions API service
 * Handles all scheduled session-related API calls
 */

import { apiClient } from './client';
import type {
  AgenticSession,
  ScheduledSession,
  CreateScheduledSessionRequest,
  UpdateScheduledSessionRequest,
} from '@/types/api';

export async function listScheduledSessions(
  projectName: string
): Promise<ScheduledSession[]> {
  const response = await apiClient.get<{ items: ScheduledSession[] }>(
    `/projects/${projectName}/scheduled-sessions`
  );
  return response.items;
}

export async function createScheduledSession(
  projectName: string,
  data: CreateScheduledSessionRequest
): Promise<ScheduledSession> {
  return apiClient.post<ScheduledSession, CreateScheduledSessionRequest>(
    `/projects/${projectName}/scheduled-sessions`,
    data
  );
}

export async function getScheduledSession(
  projectName: string,
  name: string
): Promise<ScheduledSession> {
  return apiClient.get<ScheduledSession>(
    `/projects/${projectName}/scheduled-sessions/${name}`
  );
}

export async function updateScheduledSession(
  projectName: string,
  name: string,
  data: UpdateScheduledSessionRequest
): Promise<ScheduledSession> {
  return apiClient.put<ScheduledSession, UpdateScheduledSessionRequest>(
    `/projects/${projectName}/scheduled-sessions/${name}`,
    data
  );
}

export async function deleteScheduledSession(
  projectName: string,
  name: string
): Promise<void> {
  await apiClient.delete(`/projects/${projectName}/scheduled-sessions/${name}`);
}

export async function suspendScheduledSession(
  projectName: string,
  name: string
): Promise<ScheduledSession> {
  return apiClient.post<ScheduledSession>(
    `/projects/${projectName}/scheduled-sessions/${name}/suspend`
  );
}

export async function resumeScheduledSession(
  projectName: string,
  name: string
): Promise<ScheduledSession> {
  return apiClient.post<ScheduledSession>(
    `/projects/${projectName}/scheduled-sessions/${name}/resume`
  );
}

export async function triggerScheduledSession(
  projectName: string,
  name: string
): Promise<{ name: string; namespace: string }> {
  return apiClient.post<{ name: string; namespace: string }>(
    `/projects/${projectName}/scheduled-sessions/${name}/trigger`
  );
}

export async function listScheduledSessionRuns(
  projectName: string,
  name: string
): Promise<AgenticSession[]> {
  const response = await apiClient.get<{ items: AgenticSession[] }>(
    `/projects/${projectName}/scheduled-sessions/${name}/runs`
  );
  return response.items;
}
