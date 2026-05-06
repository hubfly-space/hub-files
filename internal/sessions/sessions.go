package sessions

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"
)

var (
	ErrRateLimitExceeded = errors.New("rate limit exceeded")
	ErrMaxSessions       = errors.New("max sessions reached")
)

type Session struct {
	Code        string    `json:"sessionCode"`
	Root        string    `json:"root"`
	ExpiresAt   time.Time `json:"expiresAt"`
	ReadOnly    bool      `json:"readonly"`
	AllowUpload bool      `json:"allowUpload"`
	AllowEdit   bool      `json:"allowEdit"`
	AllowDelete bool      `json:"allowDelete"`
}

type Store struct {
	sessions    map[string]*Session
	mu          sync.RWMutex
	createLog   []time.Time
	maxSessions int
	createMu    sync.Mutex
}

func NewStore() *Store {
	store := &Store{
		sessions:    make(map[string]*Session),
		createLog:   make([]time.Time, 0),
		maxSessions: 1000,
	}
	go store.cleanupRoutine()
	return store
}

func (s *Store) CreateSession(root string, ttlSeconds int, readonly bool, allowUpload bool, allowEdit bool, allowDelete bool) (*Session, error) {
	// Rate limiting: max 10 session creations per minute
	s.createMu.Lock()
	now := time.Now()
	// Remove entries older than 1 minute
	cutoff := now.Add(-1 * time.Minute)
	valid := make([]time.Time, 0, len(s.createLog))
	for _, t := range s.createLog {
		if t.After(cutoff) {
			valid = append(valid, t)
		}
	}
	s.createLog = valid
	if len(s.createLog) >= 10 {
		s.createMu.Unlock()
		return nil, fmt.Errorf("%w: max 10 sessions per minute", ErrRateLimitExceeded)
	}
	s.createLog = append(s.createLog, now)
	s.createMu.Unlock()

	token, err := generateToken(32)
	if err != nil {
		return nil, err
	}

	session := &Session{
		Code:        token,
		Root:        root,
		ExpiresAt:   time.Now().Add(time.Duration(ttlSeconds) * time.Second),
		ReadOnly:    readonly,
		AllowUpload: allowUpload,
		AllowEdit:   allowEdit,
		AllowDelete: allowDelete,
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Check max sessions limit
	if len(s.sessions) >= s.maxSessions {
		return nil, fmt.Errorf("%w: %d", ErrMaxSessions, s.maxSessions)
	}

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
