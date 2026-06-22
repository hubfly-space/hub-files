package server

import (
	"hubfly-files/internal/sessions"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestParseSMBSessionRootSanitizesCredentials(t *testing.T) {
	root, cfg, err := parseSMBSessionRoot("smb://DOMAIN;alice:secret@example.test:1445/team/docs", "", "", "", "")
	if err != nil {
		t.Fatal(err)
	}

	if root != "smb://example.test:1445/team/docs" {
		t.Fatalf("root = %q", root)
	}
	if cfg.Host != "example.test" || cfg.Port != 1445 || cfg.Share != "team" || cfg.BasePath != "docs" {
		t.Fatalf("unexpected cfg: %+v", cfg)
	}
	if cfg.Domain != "DOMAIN" || cfg.Username != "alice" || cfg.Password != "secret" {
		t.Fatalf("unexpected credentials: domain=%q username=%q password=%q", cfg.Domain, cfg.Username, cfg.Password)
	}
}

func TestParseSMBSessionRootRejectsTraversal(t *testing.T) {
	_, _, err := parseSMBSessionRoot("smb://example.test/share/../secret", "", "", "", "")
	if err == nil {
		t.Fatal("expected traversal error")
	}
}

func TestSessionInfoReportsSMBMountCapability(t *testing.T) {
	srv, _ := newTestServer(t)
	srv.Config.AllowHostMounts = true
	srv.Config.HostMountRoot = "/mnt/hubfiles"

	session, err := srv.Sessions.CreateSMBSession("smb://fileserver/team", 3600, false, true, true, true, &sessions.SMBConfig{Host: "fileserver", Share: "team"})
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
	if !strings.Contains(rec.Body.String(), `"type":"smb"`) || !strings.Contains(rec.Body.String(), `"canHostMount":true`) {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}
