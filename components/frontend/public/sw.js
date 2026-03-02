/**
 * Service Worker for Browser Push Notifications
 * Handles push notification events and user interactions
 */

/* eslint-disable no-restricted-globals */

// Service Worker version - increment to force update
const SW_VERSION = '1.0.0';

// Install event - skip waiting to activate immediately
self.addEventListener('install', (event) => {
  console.log(`[Service Worker] Installing version ${SW_VERSION}`);
  self.skipWaiting();
});

// Activate event - claim all clients immediately
self.addEventListener('activate', (event) => {
  console.log(`[Service Worker] Activating version ${SW_VERSION}`);
  event.waitUntil(self.clients.claim());
});

// Push event - handle incoming push notifications
self.addEventListener('push', (event) => {
  console.log('[Service Worker] Push notification received');

  // Parse notification payload
  let data = {};
  if (event.data) {
    try {
      data = event.data.json();
    } catch (e) {
      console.error('[Service Worker] Failed to parse push data:', e);
      data = {
        title: 'Notification',
        body: event.data.text(),
      };
    }
  }

  // Default notification options
  const options = {
    body: data.body || 'You have a new notification',
    icon: data.icon || '/icon-192.png',
    badge: data.badge || '/badge-72.png',
    tag: data.tag || 'default',
    requireInteraction: false,
    data: data.data || {},
    actions: data.actions || [],
  };

  // Show notification
  event.waitUntil(
    self.registration.showNotification(data.title || 'Ambient Code', options)
  );
});

// Notification click event - handle user interaction
self.addEventListener('notificationclick', (event) => {
  console.log('[Service Worker] Notification clicked:', event.notification.tag);

  event.notification.close();

  // Handle action button clicks
  if (event.action) {
    console.log('[Service Worker] Action clicked:', event.action);
    // Handle specific actions (e.g., 'view', 'dismiss')
    if (event.action === 'view') {
      const urlToOpen = event.notification.data?.url || '/';
      event.waitUntil(
        clients.openWindow(urlToOpen)
      );
    }
    return;
  }

  // Default click behavior - focus or open app
  const urlToOpen = event.notification.data?.url || '/';

  event.waitUntil(
    clients
      .matchAll({ type: 'window', includeUncontrolled: true })
      .then((clientList) => {
        // Check if app is already open
        for (const client of clientList) {
          if (client.url === urlToOpen && 'focus' in client) {
            return client.focus();
          }
        }
        // Open new window if not already open
        if (clients.openWindow) {
          return clients.openWindow(urlToOpen);
        }
      })
  );
});

// Notification close event - track dismissals
self.addEventListener('notificationclose', (event) => {
  console.log('[Service Worker] Notification closed:', event.notification.tag);
  // Could send analytics here if needed
});

// Message event - handle messages from the app
self.addEventListener('message', (event) => {
  console.log('[Service Worker] Message received:', event.data);

  if (event.data && event.data.type === 'SKIP_WAITING') {
    self.skipWaiting();
  }

  if (event.data && event.data.type === 'GET_VERSION') {
    event.ports[0].postMessage({ version: SW_VERSION });
  }
});
