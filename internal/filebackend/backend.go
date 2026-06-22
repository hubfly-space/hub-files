package filebackend

import (
	"context"
	"hubfly-files/internal/filesystem"
	"io"
)

type Backend interface {
	List(ctx context.Context, subPath string) ([]filesystem.FileInfo, error)
	Storage(ctx context.Context, subPath string) (*filesystem.StorageInfo, error)
	Read(ctx context.Context, subPath string) (io.ReadCloser, error)
	Write(ctx context.Context, subPath string, data io.Reader) error
	WriteAtomic(ctx context.Context, subPath string, data io.Reader) error
	Delete(ctx context.Context, subPath string) error
	Rename(ctx context.Context, oldSubPath, newSubPath string) error
	Mkdir(ctx context.Context, subPath string) error
	Touch(ctx context.Context, subPath string) error
}
