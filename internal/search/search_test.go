package search

import (
	"path/filepath"
	"testing"
	"time"

	"hubfly-files/internal/sqlite"
)

func newTestService(t *testing.T) *Service {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "test.db")

	store, err := sqlite.New(dbPath)
	if err != nil {
		t.Fatalf("failed to create sqlite storage: %v", err)
	}

	return &Service{
		store: store,
	}
}

func TestIndexFile(t *testing.T) {
	s := newTestService(t)

	file := &sqlite.FileEntries{
		Root:       "/tmp/root",
		RelPath:    "docs/readme.md",
		BaseName:   "readme.md",
		ParentPath: "docs",
		IsDir:      false,
		Size:       123,
		ModTime:    time.Now(),
		Extension:  ".md",
	}

	if err := s.IndexFile(file); err != nil {
		t.Fatalf("IndexFile returned error: %v", err)
	}
}

func TestSearch(t *testing.T) {
	s := newTestService(t)

	files := []*sqlite.FileEntries{
		{
			Root:       "/tmp/root",
			RelPath:    "docs/readme.md",
			BaseName:   "readme.md",
			ParentPath: "docs",
			IsDir:      false,
			Size:       123,
			ModTime:    time.Now(),
			Extension:  ".md",
		},
		{
			Root:       "/tmp/root",
			RelPath:    "src/main.go",
			BaseName:   "main.go",
			ParentPath: "src",
			IsDir:      false,
			Size:       456,
			ModTime:    time.Now(),
			Extension:  ".go",
		},
	}

	for _, f := range files {
		if err := s.IndexFile(f); err != nil {
			t.Fatalf("IndexFile returned error: %v", err)
		}
	}

	results, err := s.Search("/tmp/root", "readme")
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("expected at least one search result, got 0")
	}

	found := false
	for _, r := range results {
		if r.BaseName == "readme.md" && r.RelPath == "docs/readme.md" {
			found = true
			break
		}
	}

	if !found {
		t.Fatalf("expected to find readme.md in results, got %+v", results)
	}
}

func TestSearchNoResults(t *testing.T) {
	s := newTestService(t)

	file := &sqlite.FileEntries{
		Root:       "/tmp/root",
		RelPath:    "docs/readme.md",
		BaseName:   "readme.md",
		ParentPath: "docs",
		IsDir:      false,
		Size:       123,
		ModTime:    time.Now(),
		Extension:  ".md",
	}

	if err := s.IndexFile(file); err != nil {
		t.Fatalf("IndexFile returned error: %v", err)
	}

	results, err := s.Search("/tmp/root", "something-that-does-not-exist")
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}

	if len(results) != 0 {
		t.Fatalf("expected 0 results, got %d: %+v", len(results), results)
	}
}
