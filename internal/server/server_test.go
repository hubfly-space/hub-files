package server

import (
	"bytes"
	"encoding/json"
	"hubfly-files/internal/config"
	"hubfly-files/internal/filesystem"
	"hubfly-files/internal/sessions"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
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

func createTestSession(t *testing.T, srv *Server, root string, allowUpload bool) string {
	t.Helper()

	session, err := srv.Sessions.CreateSession(root, 3600, false, allowUpload, true, true)
	if err != nil {
		t.Fatal(err)
	}

	return session.Code
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

func TestDemoStorageReturnsFakeValues(t *testing.T) {
	srv, _ := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/storage?path=/", nil)
	rec := httptest.NewRecorder()

	srv.SetupRoutes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var storage filesystem.StorageInfo
	if err := json.NewDecoder(rec.Body).Decode(&storage); err != nil {
		t.Fatalf("decode error = %v", err)
	}

	if storage.TotalBytes != DemoTotalBytes {
		t.Fatalf("totalBytes = %d, want %d", storage.TotalBytes, DemoTotalBytes)
	}
	if storage.UsedBytes != DemoUsedBytes {
		t.Fatalf("usedBytes = %d, want %d", storage.UsedBytes, DemoUsedBytes)
	}
	if storage.AvailableBytes != DemoTotalBytes-DemoUsedBytes {
		t.Fatalf("availableBytes = %d, want %d", storage.AvailableBytes, DemoTotalBytes-DemoUsedBytes)
	}
	if storage.UsedPercent != 25 {
		t.Fatalf("usedPercent = %v, want 25", storage.UsedPercent)
	}
}

func TestRawUploadAcceptsLargeFile(t *testing.T) {
	srv, tmpDir := newTestServer(t)
	token := createTestSession(t, srv, tmpDir, true)
	body := bytes.Repeat([]byte("a"), 11<<20)

	req := httptest.NewRequest(http.MethodPost, "/api/upload?path=/&filename=large.bin", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/octet-stream")
	rec := httptest.NewRecorder()

	srv.SetupRoutes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %q", rec.Code, http.StatusOK, rec.Body.String())
	}

	info, err := os.Stat(filepath.Join(tmpDir, "large.bin"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() != int64(len(body)) {
		t.Fatalf("uploaded size = %d, want %d", info.Size(), len(body))
	}
}

func TestRawUploadRejectsOversizedBody(t *testing.T) {
	srv, tmpDir := newTestServer(t)
	srv.Config.MaxUploadBytes = 4
	token := createTestSession(t, srv, tmpDir, true)

	req := httptest.NewRequest(http.MethodPost, "/api/upload?path=/&filename=too-large.bin", strings.NewReader("12345"))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/octet-stream")
	rec := httptest.NewRecorder()

	srv.SetupRoutes().ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusRequestEntityTooLarge)
	}
}

func TestRawUploadRejectsUnsafeFilename(t *testing.T) {
	srv, tmpDir := newTestServer(t)
	token := createTestSession(t, srv, tmpDir, true)

	req := httptest.NewRequest(http.MethodPost, "/api/upload?path=/&filename=../evil.txt", strings.NewReader("x"))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/octet-stream")
	rec := httptest.NewRecorder()

	srv.SetupRoutes().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestRawUploadDoesNotReplaceTargetOnInterruptedBody(t *testing.T) {
	srv, tmpDir := newTestServer(t)
	token := createTestSession(t, srv, tmpDir, true)
	targetPath := filepath.Join(tmpDir, "existing.txt")
	if err := os.WriteFile(targetPath, []byte("original"), 0644); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/upload?path=/&filename=existing.txt", errReader{})
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/octet-stream")
	rec := httptest.NewRecorder()

	srv.SetupRoutes().ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
	content, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "original" {
		t.Fatalf("target content = %q, want %q", string(content), "original")
	}
}

type errReader struct{}

func (errReader) Read([]byte) (int, error) {
	return 0, io.ErrUnexpectedEOF
}
