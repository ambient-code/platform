/**
 * Google OAuth API service (cluster-level authentication)
 */

import { apiClient } from './client';

export type GoogleAccessLevel = 'read_only' | 'full_access';

export type GoogleOAuthStatus = {
  connected: boolean;
  email?: string;
  expiresAt?: string;
  expired?: boolean;
  scopes?: string[];
  accessLevel?: GoogleAccessLevel;
};

export type GoogleOAuthURLResponse = {
  url: string;
  state: string;
  accessLevel: GoogleAccessLevel;
};

/**
 * Get Google OAuth URL for cluster-level authentication.
 * @param accessLevel - Requested Drive permission level (defaults to 'full_access')
 */
export async function getGoogleOAuthURL(
  accessLevel: GoogleAccessLevel = 'full_access'
): Promise<GoogleOAuthURLResponse> {
  return apiClient.post<GoogleOAuthURLResponse>('/auth/google/connect', { accessLevel });
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
