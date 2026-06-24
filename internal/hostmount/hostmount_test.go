package hostmount

import (
	"context"
	"hubfly-files/internal/sessions"
	"os"
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

func TestFTPMountPathIsStableAndScoped(t *testing.T) {
	cfg := sessions.FTPConfig{Host: "FTP.EXAMPLE", Port: 21, BasePath: "public/games", Username: "anonymous"}
	mountPath := FTPMountPath("/mnt/hubfiles", cfg)

	if !strings.HasPrefix(mountPath, filepath.Join("/mnt/hubfiles", "ftp-example-ftp-")) {
		t.Fatalf("mount path = %q", mountPath)
	}
	if mountPath != FTPMountPath("/mnt/hubfiles", cfg) {
		t.Fatal("mount path is not stable")
	}
}

func TestValidateFTPConfigRejectsUnsafeValues(t *testing.T) {
	cfg := sessions.FTPConfig{Host: "ftp.example", BasePath: "good,bad", Username: "alice"}
	if err := validateFTPConfig(&cfg); err == nil {
		t.Fatal("expected unsafe base path to be rejected")
	}

	cfg = sessions.FTPConfig{Host: "ftp.example", BasePath: "public", Username: "ali\nce"}
	if err := validateFTPConfig(&cfg); err == nil {
		t.Fatal("expected newline credential to be rejected")
	}
}

func TestUnmountFTPReturnsNotMounted(t *testing.T) {
	cfg := sessions.FTPConfig{Host: "ftp.example", Port: 21, BasePath: "public", Username: "anonymous"}
	result, err := UnmountFTP(context.Background(), t.TempDir(), &cfg)
	if err != nil {
		t.Fatal(err)
	}
	if result.WasMounted {
		t.Fatal("expected unmount to report not mounted")
	}
}

func TestUnmountFTPRemovesStaleMountPoint(t *testing.T) {
	mountRoot := t.TempDir()
	cfg := sessions.FTPConfig{Host: "ftp.example", Port: 21, BasePath: "public", Username: "anonymous"}
	mountPath := FTPMountPath(mountRoot, cfg)
	if err := os.MkdirAll(mountPath, 0755); err != nil {
		t.Fatal(err)
	}

	result, err := UnmountFTP(context.Background(), mountRoot, &cfg)
	if err != nil {
		t.Fatal(err)
	}
	if result.WasMounted {
		t.Fatal("expected stale mount point to report not mounted")
	}
	if _, err := os.Stat(mountPath); !os.IsNotExist(err) {
		t.Fatalf("expected stale mount point to be removed, got err=%v", err)
	}
}
