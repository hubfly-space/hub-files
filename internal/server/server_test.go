package server

import (
	"bytes"
	"encoding/json"
	"hubfly-files/internal/config"
	"hubfly-files/internal/sessions"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func newTestServer(t *testing.T) (*Server, string) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "hubfly-server-test-*")
	if err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		APIPort:        "10015",
		ManagementPort: "10014",
		DemoDir:        tmpDir,
		UIDir:          filepath.Join(tmpDir, "ui"),
	}
	if err := os.MkdirAll(cfg.UIDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfg.UIDir, "index.html"), []byte("<!doctype html>"), 0644); err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		_ = os.RemoveAll(tmpDir)
	})

	return NewServer(cfg, sessions.NewStore()), tmpDir
}

func TestAPIFileRejectsUnsupportedMethod(t *testing.T) {
	srv, _ := newTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/file?path=/test.txt", nil)
	rec := httptest.NewRecorder()

	srv.SetupRoutes().ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestGetFileReturnsNotFound(t *testing.T) {
	srv, _ := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/file?path=/missing.txt", nil)
	rec := httptest.NewRecorder()

	srv.SetupRoutes().ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestListReturnsBadRequestForInvalidPath(t *testing.T) {
	srv, _ := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/list?path=/../etc", nil)
	rec := httptest.NewRecorder()

	srv.SetupRoutes().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestCreateSessionReturnsRateLimitExceeded(t *testing.T) {
	srv, tmpDir := newTestServer(t)

	body := map[string]any{
		"root":        tmpDir,
		"ttlSeconds":  3600,
		"readonly":    false,
		"allowUpload": true,
		"allowEdit":   true,
		"allowDelete": true,
	}
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 10; i++ {
		req := httptest.NewRequest(http.MethodPost, "/sessions", bytes.NewReader(payload))
		rec := httptest.NewRecorder()
		srv.SetupManagementRoutes().ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("setup request %d status = %d, want %d", i, rec.Code, http.StatusOK)
		}
	}

	req := httptest.NewRequest(http.MethodPost, "/sessions", bytes.NewReader(payload))
	rec := httptest.NewRecorder()
	srv.SetupManagementRoutes().ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusTooManyRequests)
	}
}
