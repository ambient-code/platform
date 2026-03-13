package sessions

import "time"

type SessionMessage struct {
	ID        string    `gorm:"column:id;primaryKey;type:varchar(36)"`
	SessionID string    `gorm:"column:session_id;type:varchar(36)"`
	Seq       int64     `gorm:"column:seq"`
	EventType string    `gorm:"column:event_type;type:varchar(255)"`
	Payload   string    `gorm:"column:payload;type:text"`
	CreatedAt time.Time `gorm:"column:created_at;type:timestamptz"`
}

func (SessionMessage) TableName() string { return "session_messages" }
