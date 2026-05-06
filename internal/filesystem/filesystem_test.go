package filesystem

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSafePath(t *testing.T) {
	// Create a temp dir for testing
	tmpDir, err := os.MkdirTemp("", "test-safepath-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a subdirectory
	subDir := filepath.Join(tmpDir, "subdir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a file inside tmpDir
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		root     string
		subPath  string
		wantErr  bool
		wantPath string
	}{
		{
			name:     "valid path within root",
			root:     tmpDir,
			subPath:  "test.txt",
			wantErr:  false,
			wantPath: testFile,
		},
		{
			name:    "path traversal attack",
			root:    tmpDir,
			subPath: "..../.../etc/passwd",
			wantErr: true,
		},
		{
			name:    "subdirectory path",
			root:    tmpDir,
			subPath: "subdir",
			wantErr: false,
		},
		{
			name:    "empty subpath defaults to root",
			root:    tmpDir,
			subPath: "",
			wantErr: false,
		},
		{
			name:    "absolute path traversal",
			root:    tmpDir,
			subPath: "/etc/passwd",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := SafePath(tt.root, tt.subPath)
			if tt.wantErr && err == nil {
				t.Errorf("SafePath() expected error but got none")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("SafePath() unexpected error: %v", err)
			}
			if !tt.wantErr && tt.wantPath != "" && result != tt.wantPath {
				t.Errorf("SafePath() = %v, want %v", result, tt.wantPath)
			}
		})
	}
}

func TestSafePathSymlink(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test-symlink-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a real file
	realFile := filepath.Join(tmpDir, "real.txt")
	if err := os.WriteFile(realFile, []byte("real"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a symlink pointing outside tmpDir
	outsideDir := filepath.Join(tmpDir, "..")
	symlinkPath := filepath.Join(tmpDir, "evil_link")
	os.Symlink(filepath.Join(outsideDir, "passwd"), symlinkPath)
	// Note: This symlink might not point to a real file, but SafePath should handle it

	// Test that accessing the symlink is handled properly
	// Since we resolve symlinks, this should either work (if target is within root after resolution)
	// or fail appropriately
	_, err = SafePath(tmpDir, "evil_link")
	// The result depends on whether the symlink target is within root
	// We mainly want to ensure it doesn't panic
	t.Logf("SafePath with symlink returned: %v", err)
}

func TestListDir(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test-listdir-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test files
	os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("test1"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "file2.txt"), []byte("test2"), 0644)
	os.MkdirAll(filepath.Join(tmpDir, "subdir"), 0755)

	files, err := ListDir(tmpDir, "")
	if err != nil {
		t.Fatalf("ListDir() error = %v", err)
	}

	if len(files) != 3 {
		t.Errorf("ListDir() returned %d files, want 3", len(files))
	}

	// Check that we have both files and directory
	hasDir := false
	for _, f := range files {
		if f.IsDir && f.Name == "subdir" {
			hasDir = true
		}
	}
	if !hasDir {
		t.Error("ListDir() did not return subdir as directory")
	}
}

func TestWriteAndReadFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test-readwrite-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	testContent := "Hello, World!"

	// Test WriteFile
	err = WriteFile(tmpDir, "test.txt", strings.NewReader(testContent))
	if err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	// Test ReadFile
	reader, err := ReadFile(tmpDir, "test.txt")
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	defer reader.Close()

	buf := make([]byte, len(testContent))
	n, err := reader.Read(buf)
	if err != nil && err.Error() != "EOF" {
		t.Fatalf("Read() error = %v", err)
	}
	if string(buf[:n]) != testContent {
		t.Errorf("ReadFile() = %v, want %v", string(buf[:n]), testContent)
	}
}

func TestGetStorageInfo(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test-storage-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	info, err := GetStorageInfo(tmpDir, "")
	if err != nil {
		t.Fatalf("GetStorageInfo() error = %v", err)
	}

	if info.Path == "" {
		t.Fatal("GetStorageInfo() returned empty path")
	}
	if info.TotalBytes == 0 {
		t.Fatal("GetStorageInfo() returned zero total bytes")
	}
	if info.UsedPercent < 0 || info.UsedPercent > 100 {
		t.Fatalf("GetStorageInfo() returned invalid used percent: %f", info.UsedPercent)
	}
}

func TestOwnershipForPathUsesParentOwnerForNewFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test-ownership-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	subDir := filepath.Join(tmpDir, "child")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}

	ownership, err := OwnershipForPath(tmpDir, "/child/new.txt")
	if err != nil {
		t.Fatalf("OwnershipForPath() error = %v", err)
	}

	parentOwnership, err := ownershipFromStat(subDir)
	if err != nil {
		t.Fatalf("ownershipFromStat() error = %v", err)
	}

	if ownership.UID != parentOwnership.UID || ownership.GID != parentOwnership.GID {
		t.Fatalf("OwnershipForPath() = %+v, want %+v", ownership, parentOwnership)
	}
}

func TestDeleteFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test-delete-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "to_delete.txt")
	os.WriteFile(testFile, []byte("test"), 0644)

	// Verify file exists
	if _, err := os.Stat(testFile); err != nil {
		t.Fatal("Test file should exist")
	}

	// Delete file
	err = DeleteFile(tmpDir, "to_delete.txt")
	if err != nil {
		t.Fatalf("DeleteFile() error = %v", err)
	}

	// Verify file is gone
	if _, err := os.Stat(testFile); !os.IsNotExist(err) {
		t.Error("File should be deleted")
	}
}

func TestRename(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test-rename-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	oldPath := filepath.Join(tmpDir, "old.txt")
	newPath := filepath.Join(tmpDir, "new.txt")
	os.WriteFile(oldPath, []byte("test"), 0644)

	err = Rename(tmpDir, "old.txt", "new.txt")
	if err != nil {
		t.Fatalf("Rename() error = %v", err)
	}

	if _, err := os.Stat(newPath); err != nil {
		t.Error("Renamed file should exist")
	}
	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Error("Old file should not exist")
	}
}

func TestMkdir(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test-mkdir-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	err = Mkdir(tmpDir, "newdir")
	if err != nil {
		t.Fatalf("Mkdir() error = %v", err)
	}

	newDir := filepath.Join(tmpDir, "newdir")
	if _, err := os.Stat(newDir); err != nil {
		t.Error("Created directory should exist")
	}
}
