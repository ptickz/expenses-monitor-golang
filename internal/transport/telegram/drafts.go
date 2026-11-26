package telegram

import (
	"context"
	"sync"
	"time"
)

const draftTTL = 10 * time.Minute

type flow interface {
	handle(ctx context.Context, t *Bot, d *draft, in incoming) (done bool, err error)
}

type draftKey struct {
	chatID int64
	userID int64
}

type draft struct {
	authorID  int64
	messageID int
	timer     *time.Timer
	flow      flow
}

type drafts struct {
	mu     sync.Mutex
	byChat map[draftKey]*draft
}

func newDrafts() *drafts {
	return &drafts{byChat: make(map[draftKey]*draft)}
}

func (s *drafts) get(chatID, userID int64) *draft {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.byChat[draftKey{
		userID: userID,
		chatID: chatID,
	}]
}

func (s *drafts) start(chatID, userID int64, d *draft, onExpire func()) {
	s.mu.Lock()
	defer s.mu.Unlock()

	dKey := draftKey{
		userID: userID,
		chatID: chatID,
	}
	d.timer = time.AfterFunc(draftTTL, func() {
		s.mu.Lock()
		expired := s.byChat[dKey] == d
		if expired {
			delete(s.byChat, dKey)
		}
		s.mu.Unlock()
		if expired {
			onExpire()
		}
	})

	s.byChat[dKey] = d
}

func (s *drafts) finish(chatID, userID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	dKey := draftKey{
		userID: userID,
		chatID: chatID,
	}

	if s.byChat[dKey] != nil {
		s.byChat[dKey].timer.Stop()
		delete(s.byChat, dKey)
	}
}
