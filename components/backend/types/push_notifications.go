// Package types defines push notification related types
package types

import "time"

// PushSubscription represents a browser push notification subscription
type PushSubscription struct {
	Endpoint string               `json:"endpoint" binding:"required"`
	Keys     PushSubscriptionKeys `json:"keys" binding:"required"`
}

// PushSubscriptionKeys contains the encryption keys for push notifications
type PushSubscriptionKeys struct {
	P256dh string `json:"p256dh" binding:"required"`
	Auth   string `json:"auth" binding:"required"`
}

// NotificationPreferences defines which events trigger notifications
type NotificationPreferences struct {
	SessionStarted   bool `json:"sessionStarted"`
	SessionCompleted bool `json:"sessionCompleted"`
	SessionError     bool `json:"sessionError"`
	RunFinished      bool `json:"runFinished"`
	RunError         bool `json:"runError"`
}

// UserSubscription represents a user's push notification subscription for a project
type UserSubscription struct {
	ID           string                  `json:"id"`
	ProjectName  string                  `json:"projectName"`
	UserID       string                  `json:"userId"`
	Subscription PushSubscription        `json:"subscription"`
	Preferences  NotificationPreferences `json:"preferences"`
	CreatedAt    time.Time               `json:"createdAt"`
	UpdatedAt    time.Time               `json:"updatedAt"`
}

// CreateSubscriptionRequest is the request body for creating a push subscription
type CreateSubscriptionRequest struct {
	Subscription PushSubscription        `json:"subscription" binding:"required"`
	Preferences  NotificationPreferences `json:"preferences" binding:"required"`
}

// UpdatePreferencesRequest is the request body for updating notification preferences
type UpdatePreferencesRequest struct {
	Preferences NotificationPreferences `json:"preferences" binding:"required"`
}

// VapidPublicKeyResponse contains the VAPID public key for push notifications
type VapidPublicKeyResponse struct {
	PublicKey string `json:"publicKey"`
}

// PushNotificationPayload is the payload sent to the push service
type PushNotificationPayload struct {
	Title   string               `json:"title"`
	Body    string               `json:"body"`
	Icon    string               `json:"icon,omitempty"`
	Badge   string               `json:"badge,omitempty"`
	Tag     string               `json:"tag,omitempty"`
	Data    map[string]string    `json:"data,omitempty"`
	Actions []NotificationAction `json:"actions,omitempty"`
}

// NotificationAction represents an action button in a notification
type NotificationAction struct {
	Action string `json:"action"`
	Title  string `json:"title"`
	Icon   string `json:"icon,omitempty"`
}
