package sessions

import (
	"context"
	"fmt"
	"time"

	"github.com/ambient-code/platform/components/ambient-api-server/pkg/api"
	"github.com/openshift-online/rh-trex-ai/pkg/db"
)

type MessageDao interface {
	Insert(ctx context.Context, msg *SessionMessage) error
	AllBySessionIDAfterSeq(ctx context.Context, sessionID string, afterSeq int64) ([]SessionMessage, error)
}

var _ MessageDao = &sqlMessageDao{}

type sqlMessageDao struct {
	sessionFactory *db.SessionFactory
}

func NewMessageDao(sessionFactory *db.SessionFactory) MessageDao {
	return &sqlMessageDao{sessionFactory: sessionFactory}
}

func (d *sqlMessageDao) Insert(ctx context.Context, msg *SessionMessage) error {
	g2 := (*d.sessionFactory).New(ctx)
	msg.ID = api.NewID()
	msg.CreatedAt = time.Now().UTC()
	row := g2.Raw(
		"INSERT INTO session_messages (id, session_id, event_type, payload, created_at) VALUES (?, ?, ?, ?, ?) RETURNING seq",
		msg.ID, msg.SessionID, msg.EventType, msg.Payload, msg.CreatedAt,
	).Row()
	if err := row.Scan(&msg.Seq); err != nil {
		return fmt.Errorf("insert session message: %w", err)
	}
	return nil
}

func (d *sqlMessageDao) AllBySessionIDAfterSeq(ctx context.Context, sessionID string, afterSeq int64) ([]SessionMessage, error) {
	g2 := (*d.sessionFactory).New(ctx)
	var messages []SessionMessage
	if err := g2.Where("session_id = ? AND seq > ?", sessionID, afterSeq).Order("seq ASC").Find(&messages).Error; err != nil {
		return nil, fmt.Errorf("list session messages: %w", err)
	}
	return messages, nil
}
