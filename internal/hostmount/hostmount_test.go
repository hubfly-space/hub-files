package hostmount

import (
	"hubfly-files/internal/sessions"
	"path/filepath"
	"strings"
	"testing"
)

func TestMountPathIsStableAndScoped(t *testing.T) {
	cfg := sessions.SMBConfig{Host: "Files.EXAMPLE", Port: 445, Share: "Team Share", BasePath: "games", Username: "alice"}
	mountPath := MountPath("/mnt/hubfiles", cfg)

	if !strings.HasPrefix(mountPath, filepath.Join("/mnt/hubfiles", "files-example-team-share-")) {
		t.Fatalf("mount path = %q", mountPath)
	}
	if mountPath != MountPath("/mnt/hubfiles", cfg) {
		t.Fatal("mount path is not stable")
	}
}

func TestValidateConfigRejectsUnsafeValues(t *testing.T) {
	cfg := sessions.SMBConfig{Host: "fileserver", Share: "team", BasePath: "good,bad"}
	if err := validateConfig(&cfg); err == nil {
		t.Fatal("expected unsafe base path to be rejected")
	}

	cfg = sessions.SMBConfig{Host: "fileserver", Share: "team", Username: "ali\nce"}
	if err := validateConfig(&cfg); err == nil {
		t.Fatal("expected newline credential to be rejected")
	}
}
