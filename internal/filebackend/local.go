package filebackend

import (
	"context"
	"hubfly-files/internal/filesystem"
	"io"
)

type Local struct {
	Root string
}

func (l Local) List(_ context.Context, subPath string) ([]filesystem.FileInfo, error) {
	return filesystem.ListDir(l.Root, subPath)
}

func (l Local) Storage(_ context.Context, subPath string) (*filesystem.StorageInfo, error) {
	return filesystem.GetStorageInfo(l.Root, subPath)
}

func (l Local) Read(_ context.Context, subPath string) (io.ReadCloser, error) {
	return filesystem.ReadFile(l.Root, subPath)
}

func (l Local) Write(_ context.Context, subPath string, data io.Reader) error {
	return filesystem.WriteFile(l.Root, subPath, data)
}

func (l Local) WriteAtomic(_ context.Context, subPath string, data io.Reader) error {
	return filesystem.WriteFileAtomic(l.Root, subPath, data)
}

func (l Local) Delete(_ context.Context, subPath string) error {
	return filesystem.DeleteFile(l.Root, subPath)
}

func (l Local) Rename(_ context.Context, oldSubPath, newSubPath string) error {
	return filesystem.Rename(l.Root, oldSubPath, newSubPath)
}

func (l Local) Mkdir(_ context.Context, subPath string) error {
	return filesystem.Mkdir(l.Root, subPath)
}

func (l Local) Touch(_ context.Context, subPath string) error {
	return filesystem.Touch(l.Root, subPath)
}
