package sessions

import (
	"testing"
	"time"
)

func TestCreateAndGetSession(t *testing.T) {
	store := NewStore()

	session, err := store.CreateSession("/tmp/test", 3600, false, true, true, true)
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	if session.Code == "" {
		t.Error("CreateSession() returned session with empty Code")
	}

	if session.Root != "/tmp/test" {
		t.Errorf("Session.Root = %v, want /tmp/test", session.Root)
	}

	if !session.AllowUpload || !session.AllowEdit || !session.AllowDelete {
		t.Error("Permission flags should default to true when set")
	}

	// Test GetSession
	got, ok := store.GetSession(session.Code)
	if !ok {
		t.Error("GetSession() returned false for valid session")
	}
	if got.Code != session.Code {
		t.Errorf("GetSession() returned wrong session")
	}
}

func TestGetSessionInvalid(t *testing.T) {
	store := NewStore()

	// Test with non-existent token
	_, ok := store.GetSession("nonexistent")
	if ok {
		t.Error("GetSession() returned true for non-existent session")
	}

	// Test with empty token
	_, ok = store.GetSession("")
	if ok {
		t.Error("GetSession() returned true for empty session")
	}
}

func TestSessionExpiration(t *testing.T) {
	store := NewStore()

	// Create a session that expires in 1 second (very short TTL)
	session, err := store.CreateSession("/tmp/test", 1, false, true, true, true)
	if err != nil {
		t.Fatal(err)
	}

	// Session should be valid immediately
	_, ok := store.GetSession(session.Code)
	if !ok {
		t.Error("Session should be valid immediately after creation")
	}

	// Wait for expiration (with some buffer)
	time.Sleep(2 * time.Second)

	// Session should now be expired
	_, ok = store.GetSession(session.Code)
	if ok {
		t.Error("GetSession() should return false for expired session")
	}
}

func TestCreateSessionRateLimit(t *testing.T) {
	store := NewStore()

	// Create 10 sessions (should succeed)
	for i := 0; i < 10; i++ {
		_, err := store.CreateSession("/tmp/test", 3600, false, true, true, true)
		if err != nil {
			t.Fatalf("CreateSession() error = %v at iteration %d", err, i)
		}
	}

	// 11th session should fail due to rate limiting
	_, err := store.CreateSession("/tmp/test", 3600, false, true, true, true)
	if err == nil {
		t.Error("Expected rate limit error, got nil")
	}
}

func TestCreateSessionMaxSessions(t *testing.T) {
	store := NewStore()
	store.maxSessions = 5 // Override for testing

	// Fill up to max
	for i := 0; i < 5; i++ {
		_, err := store.CreateSession("/tmp/test", 3600, false, true, true, true)
		if err != nil {
			t.Fatalf("CreateSession() error = %v at iteration %d", err, i)
		}
	}

	// Next session should fail
	_, err := store.CreateSession("/tmp/test", 3600, false, true, true, true)
	if err == nil {
		t.Error("Expected max sessions error, got nil")
	}
}

func TestGenerateToken(t *testing.T) {
	token, err := generateToken(32)
	if err != nil {
		t.Fatalf("generateToken() error = %v", err)
	}

	if len(token) != 64 { // 32 bytes = 64 hex characters
		t.Errorf("generateToken() returned token of length %d, want 64", len(token))
	}

	// Tokens should be unique
	token2, _ := generateToken(32)
	if token == token2 {
		t.Error("generateToken() returned same token twice")
	}
}
