/**
 * Push Notification Manager
 * Handles browser Push API integration and permission management
 */

import {
  subscribeToPushNotifications,
  unsubscribeFromPushNotifications,
  getCurrentSubscription,
  getVapidPublicKey,
  type NotificationPreferences,
  type PushSubscription as PushSubscriptionType,
} from './api/push-notifications';

/**
 * Convert browser PushSubscription to our API format
 */
function serializePushSubscription(subscription: PushSubscription): PushSubscriptionType {
  const json = subscription.toJSON();
  return {
    endpoint: json.endpoint!,
    keys: {
      p256dh: json.keys!.p256dh!,
      auth: json.keys!.auth!,
    },
  };
}

/**
 * Check if push notifications are supported in the browser
 */
export function isPushNotificationSupported(): boolean {
  return (
    'serviceWorker' in navigator &&
    'PushManager' in window &&
    'Notification' in window
  );
}

/**
 * Get current notification permission status
 */
export function getNotificationPermission(): NotificationPermission {
  if (!isPushNotificationSupported()) {
    return 'denied';
  }
  return Notification.permission;
}

/**
 * Request notification permission from user
 */
export async function requestNotificationPermission(): Promise<NotificationPermission> {
  if (!isPushNotificationSupported()) {
    throw new Error('Push notifications are not supported in this browser');
  }

  const permission = await Notification.requestPermission();
  return permission;
}

/**
 * Register service worker and subscribe to push notifications
 */
export async function enablePushNotifications(
  projectName: string,
  preferences: NotificationPreferences
): Promise<void> {
  // Check browser support
  if (!isPushNotificationSupported()) {
    throw new Error('Push notifications are not supported in this browser');
  }

  // Request permission
  const permission = await requestNotificationPermission();
  if (permission !== 'granted') {
    throw new Error('Notification permission denied');
  }

  // Register service worker
  const registration = await navigator.serviceWorker.register('/sw.js', {
    scope: '/',
  });

  // Wait for service worker to be ready
  await navigator.serviceWorker.ready;

  // Get VAPID public key from backend
  const vapidPublicKey = await getVapidPublicKey();

  // Convert VAPID key to Uint8Array
  const applicationServerKey = urlBase64ToUint8Array(vapidPublicKey);

  // Subscribe to push notifications
  const subscription = await registration.pushManager.subscribe({
    userVisibleOnly: true,
    applicationServerKey,
  });

  // Send subscription to backend
  const pushSubscription = serializePushSubscription(subscription);
  await subscribeToPushNotifications(projectName, pushSubscription, preferences);
}

/**
 * Disable push notifications
 */
export async function disablePushNotifications(projectName: string): Promise<void> {
  // Get current subscription
  const currentSub = await getCurrentSubscription(projectName);
  if (!currentSub) {
    return;
  }

  // Unsubscribe from backend
  await unsubscribeFromPushNotifications(projectName, currentSub.id);

  // Unsubscribe from browser
  if ('serviceWorker' in navigator) {
    const registration = await navigator.serviceWorker.ready;
    const subscription = await registration.pushManager.getSubscription();
    if (subscription) {
      await subscription.unsubscribe();
    }
  }
}

/**
 * Check if push notifications are enabled for a project
 */
export async function isPushNotificationEnabled(projectName: string): Promise<boolean> {
  if (!isPushNotificationSupported()) {
    return false;
  }

  if (Notification.permission !== 'granted') {
    return false;
  }

  const currentSub = await getCurrentSubscription(projectName);
  return currentSub !== null;
}

/**
 * Convert base64 URL encoded string to Uint8Array
 * Required for VAPID key conversion
 */
function urlBase64ToUint8Array(base64String: string): Uint8Array {
  const padding = '='.repeat((4 - (base64String.length % 4)) % 4);
  const base64 = (base64String + padding).replace(/-/g, '+').replace(/_/g, '/');

  const rawData = window.atob(base64);
  const outputArray = new Uint8Array(rawData.length);

  for (let i = 0; i < rawData.length; ++i) {
    outputArray[i] = rawData.charCodeAt(i);
  }
  return outputArray;
}

/**
 * Show a test notification to verify setup
 */
export async function showTestNotification(): Promise<void> {
  if (!isPushNotificationSupported()) {
    throw new Error('Push notifications are not supported');
  }

  if (Notification.permission !== 'granted') {
    throw new Error('Notification permission not granted');
  }

  const registration = await navigator.serviceWorker.ready;
  await registration.showNotification('Test Notification', {
    body: 'Push notifications are working correctly!',
    icon: '/icon-192.png',
    badge: '/badge-72.png',
    tag: 'test-notification',
  });
}
