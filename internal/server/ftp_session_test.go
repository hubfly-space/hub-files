package server

import (
	"hubfly-files/internal/sessions"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestParseFTPSessionRootSanitizesCredentials(t *testing.T) {
	root, cfg, err := parseFTPSessionRoot("ftp://alice:secret@example.test:2121/public/docs", "", "")
	if err != nil {
		t.Fatal(err)
	}

	if root != "ftp://example.test:2121/public/docs" {
		t.Fatalf("root = %q", root)
	}
	if cfg.Host != "example.test" || cfg.Port != 2121 || cfg.BasePath != "public/docs" {
		t.Fatalf("unexpected cfg: %+v", cfg)
	}
	if cfg.Username != "alice" || cfg.Password != "secret" {
		t.Fatalf("unexpected credentials: %q/%q", cfg.Username, cfg.Password)
	}
}

func TestParseFTPSessionRootDefaultsAnonymousCredentials(t *testing.T) {
	_, cfg, err := parseFTPSessionRoot("ftp://example.test/public", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Username != "anonymous" || cfg.Password != "anonymous@" {
		t.Fatalf("credentials = %q/%q", cfg.Username, cfg.Password)
	}
}

func TestParseFTPSessionRootRejectsTraversal(t *testing.T) {
	_, _, err := parseFTPSessionRoot("ftp://example.test/public/../secret", "", "")
	if err == nil {
		t.Fatal("expected traversal error")
	}
}

func TestSessionInfoReportsFTP(t *testing.T) {
	srv, _ := newTestServer(t)
	srv.Config.AllowHostMounts = true
	srv.Config.HostMountRoot = "/mnt/hubfiles"
	session, err := srv.Sessions.CreateFTPSession("ftp://example.test/public", 3600, false, true, true, true, &sessions.FTPConfig{Host: "example.test", BasePath: "public", Username: "anonymous", Password: "anonymous@"})
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/session", nil)
	req.Header.Set("Authorization", "Bearer "+session.Code)
	rec := httptest.NewRecorder()

	srv.SetupRoutes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"type":"ftp"`) || !strings.Contains(body, `"canHostMount":true`) {
		t.Fatalf("unexpected body: %s", body)
	}
}
