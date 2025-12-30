/**
 * Push Notifications API client
 * Handles browser push notification subscription management
 */

import { apiClient } from './client';

export type PushSubscription = {
  endpoint: string;
  keys: {
    p256dh: string;
    auth: string;
  };
};

export type NotificationPreferences = {
  sessionStarted: boolean;
  sessionCompleted: boolean;
  sessionError: boolean;
  runFinished: boolean;
  runError: boolean;
};

export type UserSubscription = {
  id: string;
  projectName: string;
  subscription: PushSubscription;
  preferences: NotificationPreferences;
  createdAt: string;
  updatedAt: string;
};

/**
 * Subscribe to push notifications for a project
 */
export async function subscribeToPushNotifications(
  projectName: string,
  subscription: PushSubscription,
  preferences: NotificationPreferences
): Promise<UserSubscription> {
  return apiClient.post<UserSubscription>(
    `/projects/${projectName}/push-subscriptions`,
    { subscription, preferences }
  );
}

/**
 * Update notification preferences for a subscription
 */
export async function updateNotificationPreferences(
  projectName: string,
  subscriptionId: string,
  preferences: NotificationPreferences
): Promise<UserSubscription> {
  return apiClient.put<UserSubscription>(
    `/projects/${projectName}/push-subscriptions/${subscriptionId}`,
    { preferences }
  );
}

/**
 * Unsubscribe from push notifications
 */
export async function unsubscribeFromPushNotifications(
  projectName: string,
  subscriptionId: string
): Promise<void> {
  return apiClient.delete(`/projects/${projectName}/push-subscriptions/${subscriptionId}`);
}

/**
 * Get current subscription for a project
 */
export async function getCurrentSubscription(
  projectName: string
): Promise<UserSubscription | null> {
  try {
    return await apiClient.get<UserSubscription>(
      `/projects/${projectName}/push-subscriptions/current`
    );
  } catch {
    return null;
  }
}

/**
 * Get VAPID public key for push notifications
 */
export async function getVapidPublicKey(): Promise<string> {
  const response = await apiClient.get<{ publicKey: string }>('/push-notifications/vapid-public-key');
  return response.publicKey;
}
