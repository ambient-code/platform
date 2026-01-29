/**
 * Google OAuth API service (cluster-level authentication)
 */

import { apiClient } from './client';

export type GoogleOAuthStatus = {
  connected: boolean;
  email?: string;
  expiresAt?: string;
  expired?: boolean;
};

export type GoogleOAuthURLResponse = {
  url: string;
  state: string;
};

/**
 * Get Google OAuth URL for cluster-level authentication
 */
export async function getGoogleOAuthURL(): Promise<GoogleOAuthURLResponse> {
  return apiClient.post<GoogleOAuthURLResponse>('/auth/google/connect');
}

/**
 * Get Google OAuth connection status for current user
 */
export async function getGoogleStatus(): Promise<GoogleOAuthStatus> {
  return apiClient.get<GoogleOAuthStatus>('/auth/google/status');
}

/**
 * Disconnect Google OAuth for current user
 */
export async function disconnectGoogle(): Promise<void> {
  await apiClient.post('/auth/google/disconnect');
}

