package sqlite

import (
	"os"
	"testing"
	"time"
)

func newTestStorage(t *testing.T) *Storage {
	t.Helper()

	tmp := t.TempDir()

	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd() error = %v", err)
	}

	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("os.Chdir() error = %v", err)
	}

	t.Cleanup(func() {
		_ = os.Chdir(oldWd)
	})

	store, err := New("test.db")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	t.Cleanup(func() {
		_ = store.Close()
	})

	return store
}

func TestNewCreatesTables(t *testing.T) {
	store := newTestStorage(t)

	tests := []string{
		"files",
		"file_fts",
	}

	for _, tableName := range tests {
		t.Run(tableName, func(t *testing.T) {
			var name string

			err := store.db.QueryRow(`
				SELECT name
				FROM sqlite_master
				WHERE name = ?
			`, tableName).Scan(&name)

			if err != nil {
				t.Fatalf("expected table %q to exist, error = %v", tableName, err)
			}

			if name != tableName {
				t.Fatalf("expected table name %q, got %q", tableName, name)
			}
		})
	}
}

func TestRegisterFileEntry(t *testing.T) {
	store := newTestStorage(t)

	modTime := time.Date(2026, 6, 18, 10, 30, 0, 0, time.UTC)

	entry := &FileEntries{
		Root:       "/tmp/root",
		RelPath:    "docs/readme.md",
		BaseName:   "readme.md",
		ParentPath: "docs",
		IsDir:      false,
		Size:       120,
		ModTime:    modTime,
		Extension:  ".md",
	}

	result, err := store.RegisterFileEntry(entry)
	if err != nil {
		t.Fatalf("RegisterFileEntry() error = %v", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("LastInsertId() error = %v", err)
	}

	if id <= 0 {
		t.Fatalf("expected positive insert id, got %d", id)
	}

	var got FileEntries

	err = store.db.QueryRow(`
		SELECT root, rel_path, base_name, parent_path, is_dir, size, extension
		FROM files
		WHERE id = ?
	`, id).Scan(
		&got.Root,
		&got.RelPath,
		&got.BaseName,
		&got.ParentPath,
		&got.IsDir,
		&got.Size,
		&got.Extension,
	)

	if err != nil {
		t.Fatalf("query inserted file entry error = %v", err)
	}

	if got.Root != entry.Root {
		t.Fatalf("Root: expected %q, got %q", entry.Root, got.Root)
	}

	if got.RelPath != entry.RelPath {
		t.Fatalf("RelPath: expected %q, got %q", entry.RelPath, got.RelPath)
	}

	if got.BaseName != entry.BaseName {
		t.Fatalf("BaseName: expected %q, got %q", entry.BaseName, got.BaseName)
	}

	if got.ParentPath != entry.ParentPath {
		t.Fatalf("ParentPath: expected %q, got %q", entry.ParentPath, got.ParentPath)
	}

	if got.IsDir != entry.IsDir {
		t.Fatalf("IsDir: expected %v, got %v", entry.IsDir, got.IsDir)
	}

	if got.Size != entry.Size {
		t.Fatalf("Size: expected %d, got %d", entry.Size, got.Size)
	}

	if got.Extension != entry.Extension {
		t.Fatalf("Extension: expected %q, got %q", entry.Extension, got.Extension)
	}
}

func TestRegisterFileEntryUpsertsByRootAndRelPath(t *testing.T) {
	store := newTestStorage(t)

	entry := &FileEntries{
		Root:       "/tmp/root",
		RelPath:    "docs/readme.md",
		BaseName:   "readme.md",
		ParentPath: "docs",
		IsDir:      false,
		Size:       120,
		ModTime:    time.Now(),
		Extension:  ".md",
	}

	if _, err := store.RegisterFileEntry(entry); err != nil {
		t.Fatalf("first RegisterFileEntry() error = %v", err)
	}

	updated := *entry
	updated.Size = 240
	updated.BaseName = "readme-renamed.md"

	if _, err := store.RegisterFileEntry(&updated); err != nil {
		t.Fatalf("second RegisterFileEntry() error = %v", err)
	}

	otherRoot := *entry
	otherRoot.Root = "/tmp/other-root"
	if _, err := store.RegisterFileEntry(&otherRoot); err != nil {
		t.Fatalf("other root RegisterFileEntry() error = %v", err)
	}

	var count int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM files WHERE rel_path = ?`, entry.RelPath).Scan(&count); err != nil {
		t.Fatalf("count files error = %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2 root-scoped rows, got %d", count)
	}

	var gotSize int64
	var gotBaseName string
	if err := store.db.QueryRow(`
		SELECT size, base_name
		FROM files
		WHERE root = ? AND rel_path = ?
	`, entry.Root, entry.RelPath).Scan(&gotSize, &gotBaseName); err != nil {
		t.Fatalf("query updated file error = %v", err)
	}
	if gotSize != updated.Size || gotBaseName != updated.BaseName {
		t.Fatalf("expected updated row size/base_name %d/%q, got %d/%q", updated.Size, updated.BaseName, gotSize, gotBaseName)
	}
}

func TestRegisterFileIndex(t *testing.T) {
	store := newTestStorage(t)

	entry := &FileEntries{
		Root:       "/tmp/root",
		RelPath:    "docs/readme.md",
		BaseName:   "readme.md",
		ParentPath: "docs",
		IsDir:      false,
		Size:       120,
		ModTime:    time.Now(),
		Extension:  ".md",
	}

	result, err := store.RegisterFileEntry(entry)
	if err != nil {
		t.Fatalf("RegisterFileEntry() error = %v", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("LastInsertId() error = %v", err)
	}

	index := &FileIndex{
		Id:       id,
		RelPath:  entry.RelPath,
		BaseName: entry.BaseName,
	}

	if err := store.RegisterFileIndex(index); err != nil {
		t.Fatalf("RegisterFileIndex() error = %v", err)
	}

	var relPath string
	var baseName string

	err = store.db.QueryRow(`
		SELECT rel_path, base_name
		FROM file_fts
		WHERE rowid = ?
	`, id).Scan(&relPath, &baseName)

	if err != nil {
		t.Fatalf("query file_fts error = %v", err)
	}

	if relPath != entry.RelPath {
		t.Fatalf("RelPath: expected %q, got %q", entry.RelPath, relPath)
	}

	if baseName != entry.BaseName {
		t.Fatalf("BaseName: expected %q, got %q", entry.BaseName, baseName)
	}
}

func TestClose(t *testing.T) {
	tmp := t.TempDir()

	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd() error = %v", err)
	}

	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("os.Chdir() error = %v", err)
	}

	t.Cleanup(func() {
		_ = os.Chdir(oldWd)
	})

	store, err := New("test.db")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if err := store.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	if err := store.db.Ping(); err == nil {
		t.Fatal("expected Ping() after Close() to fail, got nil")
	}
}
