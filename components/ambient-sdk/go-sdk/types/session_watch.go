// SessionWatchEvent represents a real-time session change event

package types

// SessionWatchEvent represents a watch event for session changes
type SessionWatchEvent struct {
	// Type of the watch event (CREATED, UPDATED, DELETED)
	Type string `json:"type"`

	// Session object that changed (nil for DELETED events may only have metadata)
	Session *Session `json:"session,omitempty"`

	// ResourceID is the ID of the resource that changed
	ResourceID string `json:"resource_id"`
}

// IsCreated returns true if this is a creation event
func (e *SessionWatchEvent) IsCreated() bool {
	return e.Type == "CREATED"
}

// IsUpdated returns true if this is an update event
func (e *SessionWatchEvent) IsUpdated() bool {
	return e.Type == "UPDATED"
}

// IsDeleted returns true if this is a deletion event
func (e *SessionWatchEvent) IsDeleted() bool {
	return e.Type == "DELETED"
}
