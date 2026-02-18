# Browser Push Notifications

This document describes the browser push notification feature implementation for the Ambient Code Platform.

## Overview

The platform now supports browser push notifications to keep users informed of session events even when they're not actively viewing the application. Users can receive notifications for:

- Session started
- Session completed
- Session errors
- Run finished
- Run errors

## Architecture

### Components

**Frontend (TypeScript/React)**:
- `services/api/push-notifications.ts` - API client for subscription management
- `services/push-notification-manager.ts` - Browser Push API integration
- `services/queries/use-push-notifications.ts` - React Query hooks
- `public/sw.js` - Service worker for handling push events

**Backend (Go)**:
- `types/push_notifications.go` - Type definitions
- `handlers/push_notifications.go` - HTTP handlers for subscription CRUD
- `routes.go` - Route registration

### Data Flow

```
1. User enables notifications in UI
   ↓
2. Browser requests permission (Notification API)
   ↓
3. Service worker registered (/sw.js)
   ↓
4. Push subscription created (Push API)
   ↓
5. Subscription sent to backend → stored in ConfigMap
   ↓
6. Backend sends push notifications via Web Push protocol
   ↓
7. Service worker receives push → displays notification
   ↓
8. User clicks notification → app opens/focuses
```

## API Endpoints

### Public Endpoints

**GET `/api/push-notifications/vapid-public-key`**
- Returns the VAPID public key for push subscription
- No authentication required

### Project-Scoped Endpoints

All endpoints require user authentication and project access.

**POST `/api/projects/:projectName/push-subscriptions`**
- Create a new push subscription
- Body: `{ subscription: PushSubscription, preferences: NotificationPreferences }`

**GET `/api/projects/:projectName/push-subscriptions/current`**
- Get current user's subscription for the project
- Returns `null` if no subscription exists

**PUT `/api/projects/:projectName/push-subscriptions/:subscriptionId`**
- Update notification preferences
- Body: `{ preferences: NotificationPreferences }`

**DELETE `/api/projects/:projectName/push-subscriptions/:subscriptionId`**
- Delete a push subscription

## Configuration

### Backend Environment Variables

**VAPID_PUBLIC_KEY** (required)
- VAPID public key for push notifications
- Generate with: `npx web-push generate-vapid-keys`

**VAPID_PRIVATE_KEY** (required)
- VAPID private key (keep secret!)
- Used to sign push notification requests

### Frontend Environment Variables

**NEXT_PUBLIC_API_URL** (optional)
- Base URL for API requests
- Defaults to `/api` (relative path)

## Security

### Authentication
- All subscription endpoints require user authentication
- Uses user-scoped K8s clients (follows platform RBAC)
- RBAC checks via `SelfSubjectAccessReview`

### Token Handling
- VAPID keys are stored securely as environment variables
- Push subscriptions stored in namespace-scoped ConfigMaps
- User can only access their own subscriptions

### Privacy
- Subscriptions are scoped to projects and users
- No sensitive data in notification payloads
- Users can disable notifications anytime

## Browser Compatibility

| Browser | Version | Support |
|---------|---------|---------|
| Chrome | 50+ | ✅ Full |
| Firefox | 44+ | ✅ Full |
| Edge | 17+ | ✅ Full |
| Safari | 16+ | ✅ Full |
| Opera | 37+ | ✅ Full |

**Note**: Service Workers require HTTPS in production.

## User Guide

### Enabling Notifications

1. Navigate to project settings
2. Click "Enable Notifications"
3. Allow browser permission when prompted
4. Select which events trigger notifications

### Customizing Preferences

Users can toggle notifications for:
- **Session Started**: When a new session begins
- **Session Completed**: When a session finishes successfully
- **Session Error**: When a session encounters an error
- **Run Finished**: When a run completes
- **Run Error**: When a run fails

### Disabling Notifications

1. Go to project settings
2. Click "Disable Notifications"
3. Or revoke permission in browser settings

## Development

### Testing Locally

1. Ensure HTTPS is enabled (required for service workers)
2. Set VAPID keys in backend environment
3. Register service worker in development mode
4. Use browser DevTools → Application → Service Workers

### Generating VAPID Keys

```bash
npx web-push generate-vapid-keys
```

Output:
```
Public Key: BN...
Private Key: ...
```

Add to backend deployment:
```yaml
env:
  - name: VAPID_PUBLIC_KEY
    value: "BN..."
  - name: VAPID_PRIVATE_KEY
    valueFrom:
      secretKeyRef:
        name: push-notification-keys
        key: private-key
```

### Testing Push Notifications

```typescript
import { showTestNotification } from '@/services/push-notification-manager';

// Show test notification
await showTestNotification();
```

## Integration Points

### SSE Event Stream Integration

To trigger push notifications from session events, integrate with the existing AG-UI event stream in `websocket/agui.go`:

```go
// Example: Send push notification on session completion
if event.Type == "RUN_FINISHED" {
    sendPushNotification(projectName, userID, PushNotificationPayload{
        Title:   "Run Completed",
        Body:    fmt.Sprintf("Session %s finished successfully", sessionName),
        Tag:     fmt.Sprintf("session-%s", sessionName),
        Data:    map[string]string{"sessionName": sessionName, "projectName": projectName},
    })
}
```

### Operator Integration

To send notifications on session phase changes, integrate with `operator/internal/handlers/sessions.go`:

```go
// Example: Send push notification on phase transition
func updateAgenticSessionStatus(namespace, name string, updates map[string]interface{}) error {
    // ... existing status update code ...

    if phase, ok := updates["phase"].(string); ok && phase == "Completed" {
        sendPushNotificationForSession(namespace, name, "Session Completed")
    }

    return nil
}
```

## Troubleshooting

### Service Worker Not Registering

**Problem**: Service worker fails to register
**Solution**: Ensure HTTPS is enabled (required in production)

### Notifications Not Appearing

**Possible causes**:
1. Browser permission denied → Check browser settings
2. Service worker not active → Check DevTools → Application → Service Workers
3. VAPID keys not configured → Check backend environment variables
4. Subscription not created → Check network tab for API errors

### Push Subscription Fails

**Problem**: `DOMException: Registration failed`
**Solution**:
- Check VAPID public key is correctly configured
- Ensure service worker scope is `/`
- Verify HTTPS is enabled

## Future Enhancements

Potential improvements for future iterations:

1. **Web Push Library**: Integrate Go web-push library for sending notifications
2. **Batch Notifications**: Group multiple events into single notification
3. **Rich Notifications**: Add action buttons (View Session, Dismiss)
4. **Notification History**: Store notification log in backend
5. **Sound Preferences**: Allow users to customize notification sounds
6. **Desktop Integration**: Native desktop notifications on supported platforms
7. **Mobile Support**: Progressive Web App (PWA) integration
8. **Notification Analytics**: Track open rates and engagement

## References

- [Push API (MDN)](https://developer.mozilla.org/en-US/docs/Web/API/Push_API)
- [Service Worker API (MDN)](https://developer.mozilla.org/en-US/docs/Web/API/Service_Worker_API)
- [Notifications API (MDN)](https://developer.mozilla.org/en-US/docs/Web/API/Notifications_API)
- [Web Push Protocol (RFC 8030)](https://datatracker.ietf.org/doc/html/rfc8030)
- [VAPID (RFC 8292)](https://datatracker.ietf.org/doc/html/rfc8292)
