package archive

import (
	"archive/zip"
	"errors"
	"fmt"
	"hubfly-files/internal/filesystem"
	"io"
	"os"
	"path/filepath"
	"strings"
)

var (
	ErrIllegalArchivePath = errors.New("illegal archive path")
	ErrArchiveSymlink     = errors.New("archive symlinks are not allowed")
)

func Zip(source, target string, ownership *filesystem.Ownership) error {
	zipFile, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer zipFile.Close()

	if err := filesystem.ApplyOwnership(target, ownership); err != nil {
		return err
	}

	archive := zip.NewWriter(zipFile)
	defer archive.Close()

	info, err := os.Stat(source)
	if err != nil {
		return err
	}

	// Get the base directory name for relative paths
	var baseDir string
	if info.IsDir() {
		baseDir = filepath.Base(source)
	}

	err = filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Calculate relative path for zip entry
		var entryPath string
		if baseDir != "" {
			// For directories, we want to include the base directory
			relPath, err := filepath.Rel(source, path)
			if err != nil {
				return err
			}
			entryPath = filepath.Join(baseDir, relPath)
		} else {
			entryPath = filepath.Base(path)
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		header.Name = entryPath

		if info.IsDir() {
			header.Name += "/"
		} else {
			header.Method = zip.Deflate
		}

		writer, err := archive.CreateHeader(header)
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()
		_, err = io.Copy(writer, file)
		return err
	})

	return err
}

func Unzip(source, target string, ownership *filesystem.Ownership) error {
	reader, err := zip.OpenReader(source)
	if err != nil {
		return err
	}
	defer reader.Close()

	for _, f := range reader.File {
		// Reject symlinks to prevent ZipSlip variant attacks
		if f.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("%w: %s", ErrArchiveSymlink, f.Name)
		}

		path := filepath.Join(target, f.Name)

		// Check for ZipSlip vulnerability
		if !strings.HasPrefix(path, filepath.Clean(target)+string(os.PathSeparator)) {
			return fmt.Errorf("%w: %s", ErrIllegalArchivePath, path)
		}

		if f.FileInfo().IsDir() {
			if err := filesystem.CreateDirAllWithOwnership(path, 0755, ownership); err != nil {
				return err
			}
			continue
		}

		if err := filesystem.CreateDirAllWithOwnership(filepath.Dir(path), 0755, ownership); err != nil {
			return err
		}

		// Use safe permissions instead of preserving from zip
		var fileMode os.FileMode = 0644
		outFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, fileMode)
		if err != nil {
			return err
		}

		if err := filesystem.ApplyOwnership(path, ownership); err != nil {
			outFile.Close()
			return err
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)

		outFile.Close()
		rc.Close()

		if err != nil {
			return err
		}
	}
	return nil
}
