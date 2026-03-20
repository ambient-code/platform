package sessions

import (
	"context"
	"fmt"
	"sync"
)

type MessageService interface {
	Push(ctx context.Context, sessionID, eventType, payload string) (*SessionMessage, error)
	Subscribe(ctx context.Context, sessionID string) (<-chan *SessionMessage, func())
	AllBySessionIDAfterSeq(ctx context.Context, sessionID string, afterSeq int64) ([]SessionMessage, error)
}

type sqlMessageService struct {
	dao  MessageDao
	mu   sync.RWMutex
	subs map[string][]chan *SessionMessage
}

func NewMessageService(dao MessageDao) MessageService {
	return &sqlMessageService{
		dao:  dao,
		subs: make(map[string][]chan *SessionMessage),
	}
}

func (s *sqlMessageService) Push(ctx context.Context, sessionID, eventType, payload string) (*SessionMessage, error) {
	msg := &SessionMessage{
		SessionID: sessionID,
		EventType: eventType,
		Payload:   payload,
	}
	if err := s.dao.Insert(ctx, msg); err != nil {
		return nil, fmt.Errorf("push session message: %w", err)
	}

	s.mu.RLock()
	chans := make([]chan *SessionMessage, len(s.subs[sessionID]))
	copy(chans, s.subs[sessionID])
	s.mu.RUnlock()

	for _, ch := range chans {
		select {
		case ch <- msg:
		default:
		}
	}
	return msg, nil
}

func (s *sqlMessageService) Subscribe(ctx context.Context, sessionID string) (<-chan *SessionMessage, func()) {
	ch := make(chan *SessionMessage, 512)

	s.mu.Lock()
	s.subs[sessionID] = append(s.subs[sessionID], ch)
	s.mu.Unlock()

	var once sync.Once
	remove := func() {
		once.Do(func() {
			s.mu.Lock()
			defer s.mu.Unlock()
			subs := s.subs[sessionID]
			for i, sub := range subs {
				if sub == ch {
					s.subs[sessionID] = append(subs[:i], subs[i+1:]...)
					close(ch)
					return
				}
			}
		})
	}

	go func() {
		<-ctx.Done()
		remove()
	}()

	return ch, remove
}

func (s *sqlMessageService) AllBySessionIDAfterSeq(ctx context.Context, sessionID string, afterSeq int64) ([]SessionMessage, error) {
	return s.dao.AllBySessionIDAfterSeq(ctx, sessionID, afterSeq)
}
