package types

import "time"

type SessionMessage struct {
	ID        string    `json:"id"`
	SessionID string    `json:"session_id"`
	Seq       int64     `json:"seq"`
	EventType string    `json:"event_type"`
	Payload   string    `json:"payload"`
	CreatedAt time.Time `json:"created_at"`
}

type SessionMessagePush struct {
	EventType string `json:"event_type"`
	Payload   string `json:"payload"`
}
