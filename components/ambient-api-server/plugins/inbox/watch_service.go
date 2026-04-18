package inbox

import (
	"context"
	"sync"
)

type InboxWatchService interface {
	Subscribe(ctx context.Context, agentID string) (<-chan *InboxMessage, func())
	Notify(msg *InboxMessage)
}

type inboxWatchService struct {
	mu   sync.RWMutex
	subs map[string][]chan *InboxMessage
}

func NewInboxWatchService() InboxWatchService {
	return &inboxWatchService{
		subs: make(map[string][]chan *InboxMessage),
	}
}

func (s *inboxWatchService) Subscribe(ctx context.Context, agentID string) (<-chan *InboxMessage, func()) {
	ch := make(chan *InboxMessage, 512)

	s.mu.Lock()
	s.subs[agentID] = append(s.subs[agentID], ch)
	s.mu.Unlock()

	var once sync.Once
	remove := func() {
		once.Do(func() {
			s.mu.Lock()
			defer s.mu.Unlock()
			subs := s.subs[agentID]
			for i, sub := range subs {
				if sub == ch {
					s.subs[agentID] = append(subs[:i], subs[i+1:]...)
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

func (s *inboxWatchService) Notify(msg *InboxMessage) {
	s.mu.RLock()
	chans := make([]chan *InboxMessage, len(s.subs[msg.AgentId]))
	copy(chans, s.subs[msg.AgentId])
	s.mu.RUnlock()

	for _, ch := range chans {
		func() {
			defer func() { _ = recover() }()
			select {
			case ch <- msg:
			default:
			}
		}()
	}
}
