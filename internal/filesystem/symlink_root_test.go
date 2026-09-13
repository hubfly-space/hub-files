package filesystem

import (
	"os"
	"path/filepath"
	"testing"
)

// A session root that itself lives beneath a symlink (e.g. container
// mounts like /var/lib/hubcell -> /data/hubcell) must still work.
func TestSafePathRootUnderSymlink(t *testing.T) {
	realRoot, err := os.MkdirTemp("", "test-symlink-root-real-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(realRoot)

	link := filepath.Join(filepath.Dir(realRoot), "symlink-root-view-"+filepath.Base(realRoot))
	os.Remove(link)
	defer os.Remove(link)

	if err := os.Symlink(realRoot, link); err != nil {
		t.Fatal(err)
	}

	root := filepath.Join(link, "sub")
	if err := os.MkdirAll(root, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := SafePath(root, "")
	if err != nil {
		t.Fatalf("SafePath(root, \"\") unexpected error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(result, "a.txt")); err != nil {
		t.Fatalf("expected a.txt reachable under symlinked root: %v", err)
	}

	result, err = SafePath(root, "a.txt")
	if err != nil {
		t.Fatalf("SafePath(root, \"a.txt\") unexpected error: %v", err)
	}
	if err := os.WriteFile(result, []byte("y"), 0644); err != nil {
		t.Fatalf("write through symlinked root failed: %v", err)
	}
}