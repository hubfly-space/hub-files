package filesystem

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
)

var (
	ErrUnauthorized = errors.New("unauthorized access")
)

type FileInfo struct {
	Name    string `json:"name"`
	IsDir   bool   `json:"isDir"`
	Size    int64  `json:"size"`
	ModTime string `json:"modTime"`
}

func SafePath(root, subPath string) (string, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}

	finalPath := filepath.Join(absRoot, filepath.FromSlash(subPath))
	absFinal, err := filepath.Abs(finalPath)
	if err != nil {
		return "", err
	}

	if !strings.HasPrefix(absFinal, absRoot) {
		return "", ErrUnauthorized
	}

	// Resolve symlinks to prevent symlink traversal attacks
	resolved, err := filepath.EvalSymlinks(absFinal)
	if err != nil {
		// If file doesn't exist yet (for write operations), check parent
		parent := filepath.Dir(absFinal)
		resolvedParent, err2 := filepath.EvalSymlinks(parent)
		if err2 != nil {
			return "", ErrUnauthorized
		}
		if !strings.HasPrefix(resolvedParent, absRoot) {
			return "", ErrUnauthorized
		}
		return absFinal, nil
	}

	if !strings.HasPrefix(resolved, absRoot) {
		return "", ErrUnauthorized
	}

	return resolved, nil
}

func ListDir(root, subPath string) ([]FileInfo, error) {
	path, err := SafePath(root, subPath)
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	files := make([]FileInfo, 0, len(entries))
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}
		files = append(files, FileInfo{
			Name:    entry.Name(),
			IsDir:   entry.IsDir(),
			Size:    info.Size(),
			ModTime: info.ModTime().Format("2006-01-02 15:04:05"),
		})
	}

	return files, nil
}

func ReadFile(root, subPath string) (io.ReadCloser, error) {
	path, err := SafePath(root, subPath)
	if err != nil {
		return nil, err
	}
	return os.Open(path)
}

func WriteFile(root, subPath string, data io.Reader) error {
	path, err := SafePath(root, subPath)
	if err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, data)
	return err
}

func DeleteFile(root, subPath string) error {
	path, err := SafePath(root, subPath)
	if err != nil {
		return err
	}
	return os.RemoveAll(path)
}

func Rename(root, oldSub, newSub string) error {
	oldPath, err := SafePath(root, oldSub)
	if err != nil {
		return err
	}
	newPath, err := SafePath(root, newSub)
	if err != nil {
		return err
	}
	return os.Rename(oldPath, newPath)
}

func Mkdir(root, subPath string) error {
	path, err := SafePath(root, subPath)
	if err != nil {
		return err
	}
	return os.MkdirAll(path, 0755)
}
