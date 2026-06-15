package filesystem

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
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

func TestSafePathRejectsSiblingWithSharedPrefix(t *testing.T) {
	parent, err := os.MkdirTemp("", "test-safepath-prefix-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(parent)

	root := filepath.Join(parent, "root")
	sibling := filepath.Join(parent, "root-sibling")
	if err := os.MkdirAll(root, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(sibling, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sibling, "evil.txt"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err = SafePath(root, "../root-sibling/evil.txt")
	if err == nil {
		t.Fatal("SafePath() expected shared-prefix sibling to be rejected")
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

func TestWriteFileAtomicKeepsExistingTargetOnReadError(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test-atomic-write-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	if err := os.WriteFile(filepath.Join(tmpDir, "target.txt"), []byte("original"), 0644); err != nil {
		t.Fatal(err)
	}

	err = WriteFileAtomic(tmpDir, "target.txt", failingReader{})
	if err == nil {
		t.Fatal("WriteFileAtomic() expected error")
	}

	content, err := os.ReadFile(filepath.Join(tmpDir, "target.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "original" {
		t.Fatalf("target content = %q, want %q", string(content), "original")
	}

	matches, err := filepath.Glob(filepath.Join(tmpDir, ".*.upload"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("temporary upload files left behind: %v", matches)
	}
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) {
	return 0, os.ErrPermission
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

func testOwnershipFromPath(t *testing.T, path string) *Ownership {
	t.Helper()

	ownership, err := ownershipFromStat(path)
	if err != nil {
		t.Fatalf("ownershipFromStat() error = %v", err)
	}

	return ownership
}

func TestCreateFileWithAllOwnershipCreatesFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test-create-file-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	ownership := testOwnershipFromPath(t, tmpDir)

	filePath := filepath.Join(tmpDir, "uploads", "images", "avatar.txt")

	err = CreateFileWithAllOwnership(filePath, 0644, ownership)
	if err != nil {
		t.Fatalf("CreateFileWithAllOwnership() error = %v", err)
	}

	info, err := os.Stat(filePath)
	if err != nil {
		t.Fatalf("created file does not exist: %v", err)
	}

	if info.IsDir() {
		t.Fatal("created path is a directory, want file")
	}

	parent := filepath.Dir(filePath)
	parentInfo, err := os.Stat(parent)
	if err != nil {
		t.Fatalf("parent directory was not created: %v", err)
	}

	if !parentInfo.IsDir() {
		t.Fatal("parent path exists but is not a directory")
	}
}

func TestCreateFileWithAllOwnershipExistingFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test-create-existing-file-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	ownership := testOwnershipFromPath(t, tmpDir)

	filePath := filepath.Join(tmpDir, "existing.txt")
	originalContent := []byte("do not destroy me")

	if err := os.WriteFile(filePath, originalContent, 0644); err != nil {
		t.Fatal(err)
	}

	err = CreateFileWithAllOwnership(filePath, 0644, ownership)
	if err != nil {
		t.Fatalf("CreateFileWithAllOwnership() error = %v", err)
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatal(err)
	}

	if string(content) != string(originalContent) {
		t.Fatalf("file content changed = %q, want %q", string(content), string(originalContent))
	}
}

func TestCreateFileWithAllOwnershipRejectsDirectory(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test-create-file-reject-dir-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	ownership := testOwnershipFromPath(t, tmpDir)

	dirPath := filepath.Join(tmpDir, "already-a-dir")
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		t.Fatal(err)
	}

	err = CreateFileWithAllOwnership(dirPath, 0644, ownership)
	if err == nil {
		t.Fatal("CreateFileWithAllOwnership() expected error for directory path")
	}

	var pathErr *os.PathError
	if !errors.As(err, &pathErr) {
		t.Fatalf("error type = %T, want *os.PathError", err)
	}

	if !errors.Is(pathErr.Err, syscall.EISDIR) {
		t.Fatalf("PathError.Err = %v, want %v", pathErr.Err, syscall.EISDIR)
	}
}

func TestCreateFileWithAllOwnershipParentComponentIsFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test-create-file-parent-is-file-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	ownership := testOwnershipFromPath(t, tmpDir)

	blocker := filepath.Join(tmpDir, "blocker")
	if err := os.WriteFile(blocker, []byte("I am a file, not a folder"), 0644); err != nil {
		t.Fatal(err)
	}

	filePath := filepath.Join(blocker, "child.txt")

	err = CreateFileWithAllOwnership(filePath, 0644, ownership)
	if err == nil {
		t.Fatal("CreateFileWithAllOwnership() expected error when parent component is a file")
	}
}

func TestCreateFileWithAllOwnershipCleansPath(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test-create-file-clean-path-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	ownership := testOwnershipFromPath(t, tmpDir)

	dirtyPath := filepath.Join(tmpDir, "uploads", ".", "images", "..", "file.txt")
	cleanPath := filepath.Clean(dirtyPath)

	err = CreateFileWithAllOwnership(dirtyPath, 0644, ownership)
	if err != nil {
		t.Fatalf("CreateFileWithAllOwnership() error = %v", err)
	}

	if _, err := os.Stat(cleanPath); err != nil {
		t.Fatalf("cleaned file path does not exist: %v", err)
	}
}
