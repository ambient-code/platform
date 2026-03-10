/**
 * Scheduled Session API types
 * These types align with the backend Go structs
 */

import type { CreateAgenticSessionRequest } from './sessions';

export type ScheduledSession = {
  name: string;
  namespace: string;
  creationTimestamp: string;
  schedule: string;
  suspend: boolean;
  displayName: string;
  sessionTemplate: CreateAgenticSessionRequest;
  lastScheduleTime?: string;
  activeCount: number;
  labels?: Record<string, string>;
  annotations?: Record<string, string>;
};

export type CreateScheduledSessionRequest = {
  displayName?: string;
  schedule: string;
  sessionTemplate: CreateAgenticSessionRequest;
  suspend?: boolean;
};

export type UpdateScheduledSessionRequest = {
  displayName?: string;
  schedule?: string;
  sessionTemplate?: CreateAgenticSessionRequest;
  suspend?: boolean;
};
