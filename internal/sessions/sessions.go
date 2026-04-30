package sessions

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

type Session struct {
	Code      string    `json:"sessionCode"`
	Root      string    `json:"root"`
	ExpiresAt time.Time `json:"expiresAt"`
	ReadOnly  bool      `json:"readonly"`
}

type Store struct {
	sessions map[string]*Session
	mu       sync.RWMutex
}

func NewStore() *Store {
	store := &Store{
		sessions: make(map[string]*Session),
	}
	go store.cleanupRoutine()
	return store
}

func (s *Store) CreateSession(root string, ttlSeconds int, readonly bool) (*Session, error) {
	token, err := generateToken(32)
	if err != nil {
		return nil, err
	}

	session := &Session{
		Code:      token,
		Root:      root,
		ExpiresAt: time.Now().Add(time.Duration(ttlSeconds) * time.Second),
		ReadOnly:  readonly,
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[token] = session

	return session, nil
}

func (s *Store) GetSession(token string) (*Session, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, ok := s.sessions[token]
	if !ok {
		return nil, false
	}

	if time.Now().After(session.ExpiresAt) {
		return nil, false
	}

	return session, true
}

func (s *Store) cleanupRoutine() {
	ticker := time.NewTicker(1 * time.Minute)
	for range ticker.C {
		s.mu.Lock()
		for token, session := range s.sessions {
			if time.Now().After(session.ExpiresAt) {
				delete(s.sessions, token)
			}
		}
		s.mu.Unlock()
	}
}

func generateToken(length int) (string, error) {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
