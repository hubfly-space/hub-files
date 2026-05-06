package archive

import (
	"archive/zip"
	"hubfly-files/internal/filesystem"
	"os"
	"path/filepath"
	"testing"
)

func TestZipAndUnzip(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test-archive-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test directory with files
	sourceDir := filepath.Join(tmpDir, "source")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatal(err)
	}

	testFile1 := filepath.Join(sourceDir, "file1.txt")
	os.WriteFile(testFile1, []byte("content1"), 0644)

	testFile2 := filepath.Join(sourceDir, "file2.txt")
	os.WriteFile(testFile2, []byte("content2"), 0644)

	subDir := filepath.Join(sourceDir, "subdir")
	os.MkdirAll(subDir, 0755)
	os.WriteFile(filepath.Join(subDir, "file3.txt"), []byte("content3"), 0644)

	// Create zip
	zipPath := filepath.Join(tmpDir, "test.zip")
	err = Zip(sourceDir, zipPath, &filesystem.Ownership{})
	if err != nil {
		t.Fatalf("Zip() error = %v", err)
	}

	// Verify zip file exists
	if _, err := os.Stat(zipPath); err != nil {
		t.Error("Zip() did not create output file")
	}

	// Extract zip
	extractDir := filepath.Join(tmpDir, "extracted")
	err = Unzip(zipPath, extractDir, &filesystem.Ownership{})
	if err != nil {
		t.Fatalf("Unzip() error = %v", err)
	}

	// Debug: list what's in the zip
	readZip, _ := zip.OpenReader(zipPath)
	for _, f := range readZip.File {
		t.Logf("Zip contains: %s", f.Name)
	}
	readZip.Close()

	// Debug: list what's extracted
	filepath.Walk(extractDir, func(path string, info os.FileInfo, err error) error {
		t.Logf("Extracted: %s", path)
		return nil
	})

	// Verify extracted files (they will be under 'source/' directory)
	if _, err := os.Stat(filepath.Join(extractDir, "source", "file1.txt")); err != nil {
		t.Errorf("Unzip() did not extract file1.txt: %v", err)
	}
	if _, err := os.Stat(filepath.Join(extractDir, "source", "file2.txt")); err != nil {
		t.Errorf("Unzip() did not extract file2.txt: %v", err)
	}
	if _, err := os.Stat(filepath.Join(extractDir, "source", "subdir", "file3.txt")); err != nil {
		t.Errorf("Unzip() did not extract file3.txt in subdirectory: %v", err)
	}
}

func TestZipSlipProtection(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test-zipslip-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a zip file with path traversal in the name
	maliciousZip := filepath.Join(tmpDir, "malicious.zip")
	zipFile, err := os.Create(maliciousZip)
	if err != nil {
		t.Fatal(err)
	}

	writer := zip.NewWriter(zipFile)

	// Create a header with path traversal
	header := &zip.FileHeader{
		Name: "../../../etc/passwd",
	}
	w, err := writer.CreateHeader(header)
	if err != nil {
		t.Fatal(err)
	}
	w.Write([]byte("malicious"))
	writer.Close()
	zipFile.Close()

	// Try to unzip - should fail with illegal file path error
	extractDir := filepath.Join(tmpDir, "extract")
	err = Unzip(maliciousZip, extractDir, &filesystem.Ownership{})
	if err == nil {
		t.Error("Unzip() should fail for path traversal attempts")
	}
}

func TestUnzipSymlinkRejection(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test-symlink-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a zip file with a symlink entry
	symlinkZip := filepath.Join(tmpDir, "symlink.zip")
	zipFile, err := os.Create(symlinkZip)
	if err != nil {
		t.Fatal(err)
	}

	writer := zip.NewWriter(zipFile)

	// Create a file header with ModeSymlink
	header := &zip.FileHeader{
		Name: "symlink.txt",
	}
	header.SetMode(os.ModeSymlink)
	w, err := writer.CreateHeader(header)
	if err != nil {
		t.Fatal(err)
	}
	w.Write([]byte("/etc/passwd"))
	writer.Close()
	zipFile.Close()

	// Try to unzip - should fail because symlinks are rejected
	extractDir := filepath.Join(tmpDir, "extract")
	err = Unzip(symlinkZip, extractDir, &filesystem.Ownership{})
	if err == nil {
		t.Error("Unzip() should reject symlinks")
	}
}
