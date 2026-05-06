package filesystem

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
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

type StorageInfo struct {
	Path           string  `json:"path"`
	TotalBytes     uint64  `json:"totalBytes"`
	UsedBytes      uint64  `json:"usedBytes"`
	AvailableBytes uint64  `json:"availableBytes"`
	UsedPercent    float64 `json:"usedPercent"`
}

type Ownership struct {
	UID int
	GID int
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

func GetStorageInfo(root, subPath string) (*StorageInfo, error) {
	path, err := SafePath(root, subPath)
	if err != nil {
		return nil, err
	}

	var stats syscall.Statfs_t
	if err := syscall.Statfs(path, &stats); err != nil {
		return nil, err
	}

	blockSize := uint64(stats.Bsize)
	total := stats.Blocks * blockSize
	available := stats.Bavail * blockSize
	used := (stats.Blocks - stats.Bfree) * blockSize

	usedPercent := 0.0
	if total > 0 {
		usedPercent = (float64(used) / float64(total)) * 100
	}

	return &StorageInfo{
		Path:           path,
		TotalBytes:     total,
		UsedBytes:      used,
		AvailableBytes: available,
		UsedPercent:    usedPercent,
	}, nil
}

func ownershipFromStat(path string) (*Ownership, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return nil, errors.New("unsupported file stat data")
	}

	return &Ownership{
		UID: int(stat.Uid),
		GID: int(stat.Gid),
	}, nil
}

func nearestExistingOwnership(path string) (*Ownership, error) {
	current := filepath.Clean(path)
	for {
		info, err := os.Stat(current)
		if err == nil {
			if info.IsDir() {
				return ownershipFromStat(current)
			}
			return ownershipFromStat(filepath.Dir(current))
		}
		if !os.IsNotExist(err) {
			return nil, err
		}

		next := filepath.Dir(current)
		if next == current {
			break
		}
		current = next
	}

	return nil, os.ErrNotExist
}

func DesiredOwnershipForPath(path string) (*Ownership, error) {
	if _, err := os.Stat(path); err == nil {
		return ownershipFromStat(path)
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	return nearestExistingOwnership(filepath.Dir(path))
}

func OwnershipForPath(root, subPath string) (*Ownership, error) {
	path, err := SafePath(root, subPath)
	if err != nil {
		return nil, err
	}
	return DesiredOwnershipForPath(path)
}

func ApplyOwnership(path string, ownership *Ownership) error {
	if ownership == nil || os.Geteuid() != 0 {
		return nil
	}
	return os.Chown(path, ownership.UID, ownership.GID)
}

func CreateDirAllWithOwnership(path string, perm os.FileMode, ownership *Ownership) error {
	cleanPath := filepath.Clean(path)
	if info, err := os.Stat(cleanPath); err == nil {
		if info.IsDir() {
			return nil
		}
		return &os.PathError{Op: "mkdir", Path: cleanPath, Err: syscall.ENOTDIR}
	} else if !os.IsNotExist(err) {
		return err
	}

	parent := filepath.Dir(cleanPath)
	if parent != cleanPath {
		if err := CreateDirAllWithOwnership(parent, perm, ownership); err != nil {
			return err
		}
	}

	if err := os.Mkdir(cleanPath, perm); err != nil {
		if os.IsExist(err) {
			return nil
		}
		return err
	}

	return ApplyOwnership(cleanPath, ownership)
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

	ownership, err := DesiredOwnershipForPath(path)
	if err != nil {
		return err
	}

	_, statErr := os.Stat(path)
	fileExists := statErr == nil
	if statErr != nil && !os.IsNotExist(statErr) {
		return statErr
	}

	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	if !fileExists {
		if err := ApplyOwnership(path, ownership); err != nil {
			return err
		}
	}

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

	ownership, err := DesiredOwnershipForPath(path)
	if err != nil {
		return err
	}

	return CreateDirAllWithOwnership(path, 0755, ownership)
}
