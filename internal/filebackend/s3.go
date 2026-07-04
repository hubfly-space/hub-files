package filebackend

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"hubfly-files/internal/filesystem"
	"hubfly-files/internal/sessions"
	"io"
	"path"
	"sort"
	"strings"
	"sync"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

const s3ListPageSize = 1000

type S3Pool struct {
	mu      sync.Mutex
	clients map[string]*minio.Client
}

type S3 struct {
	Pool   *S3Pool
	Key    string
	Config sessions.S3Config
}

func NewS3Pool() *S3Pool {
	return &S3Pool{clients: make(map[string]*minio.Client)}
}

func (p *S3Pool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.clients = make(map[string]*minio.Client)
	return nil
}

func (p *S3Pool) Invalidate(key string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.clients, key)
}

func (p *S3Pool) getClient(ctx context.Context, key string, cfg sessions.S3Config) (*minio.Client, error) {
	p.mu.Lock()
	if client, ok := p.clients[key]; ok {
		p.mu.Unlock()
		return client, nil
	}
	p.mu.Unlock()

	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
		Region: cfg.Region,
	})
	if err != nil {
		return nil, err
	}

	p.mu.Lock()
	if existing, ok := p.clients[key]; ok {
		p.mu.Unlock()
		return existing, nil
	}
	p.clients[key] = client
	p.mu.Unlock()

	// Verify the bucket exists
	exists, err := client.BucketExists(ctx, cfg.Bucket)
	if err != nil {
		p.Invalidate(key)
		return nil, fmt.Errorf("s3: cannot access bucket %s: %w", cfg.Bucket, err)
	}
	if !exists {
		p.Invalidate(key)
		return nil, fmt.Errorf("s3: bucket %s does not exist", cfg.Bucket)
	}

	return client, nil
}

func (s S3) withClient(ctx context.Context, fn func(*minio.Client) error) error {
	client, err := s.Pool.getClient(ctx, s.Key, s.Config)
	if err != nil {
		return err
	}
	if err := fn(client); err == nil {
		return nil
	}
	s.Pool.Invalidate(s.Key)
	client, err = s.Pool.getClient(ctx, s.Key, s.Config)
	if err != nil {
		return err
	}
	return fn(client)
}

func (s S3) fullPath(subPath string) string {
	cleaned := cleanS3Path(subPath)
	base := strings.Trim(s.Config.BasePath, "/")
	if base == "" {
		return cleaned
	}
	return path.Join(base, cleaned)
}

func cleanS3Path(p string) string {
	p = strings.ReplaceAll(p, "\\", "/")
	parts := strings.Split(p, "/")
	cleaned := make([]string, 0, len(parts))
	for _, part := range parts {
		switch part {
		case "", ".":
			continue
		case "..":
			// Reject traversal
			return ""
		default:
			cleaned = append(cleaned, part)
		}
	}
	return strings.Join(cleaned, "/")
}

func pathIsDir(key string) bool {
	return strings.HasSuffix(key, "/")
}

func dirKey(key string) string {
	if key == "" || strings.HasSuffix(key, "/") {
		return key
	}
	return key + "/"
}

func (s S3) List(ctx context.Context, subPath string) ([]filesystem.FileInfo, error) {
	prefix := s.fullPath(subPath)
	if prefix != "" {
		prefix += "/"
	}

	var files []filesystem.FileInfo

	err := s.withClient(ctx, func(client *minio.Client) error {
		opts := minio.ListObjectsOptions{
			Prefix:    prefix,
			Recursive: false,
			MaxKeys:   s3ListPageSize,
		}

		seen := make(map[string]bool)

		for obj := range client.ListObjects(ctx, s.Config.Bucket, opts) {
			if obj.Err != nil {
				return obj.Err
			}

			// Common prefixes (directories) have only Key set, ending with "/"
			// Regular objects have full metadata.
			isDir := strings.HasSuffix(obj.Key, "/")

			name := obj.Key
			if prefix != "" {
				name = strings.TrimPrefix(obj.Key, prefix)
			}
			name = strings.TrimSuffix(name, "/")

			if name == "" || seen[name] {
				continue
			}
			seen[name] = true

			modTime := ""
			if !isDir && !obj.LastModified.IsZero() {
				modTime = obj.LastModified.Format("2006-01-02 15:04:05")
			}

			files = append(files, filesystem.FileInfo{
				Name:    name,
				IsDir:   isDir,
				Size:    obj.Size,
				ModTime: modTime,
			})
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	// Sort by name for stable listing
	sort.Slice(files, func(i, j int) bool {
		if files[i].IsDir != files[j].IsDir {
			return files[i].IsDir // dirs first
		}
		return files[i].Name < files[j].Name
	})

	return files, nil
}

func (s S3) Storage(ctx context.Context, _ string) (*filesystem.StorageInfo, error) {
	// S3 doesn't provide a simple "disk usage" API across all providers.
	// Return a best-effort estimate.
	return &filesystem.StorageInfo{
		Path:           "/",
		TotalBytes:     0,
		UsedBytes:      0,
		AvailableBytes: 0,
		UsedPercent:    0,
	}, nil
}

func (s S3) Read(ctx context.Context, subPath string) (io.ReadCloser, error) {
	remotePath := s.fullPath(subPath)
	if remotePath == "" {
		return nil, filesystem.ErrUnauthorized
	}

	var reader io.ReadCloser
	err := s.withClient(ctx, func(client *minio.Client) error {
		obj, err := client.GetObject(ctx, s.Config.Bucket, remotePath, minio.GetObjectOptions{})
		if err != nil {
			return err
		}
		reader = obj
		return nil
	})
	if err != nil {
		return nil, err
	}
	return reader, nil
}

func (s S3) Write(ctx context.Context, subPath string, data io.Reader) error {
	return s.write(ctx, subPath, data)
}

func (s S3) WriteAtomic(ctx context.Context, subPath string, data io.Reader) error {
	return s.write(ctx, subPath, data)
}

func (s S3) write(ctx context.Context, subPath string, data io.Reader) error {
	remotePath := s.fullPath(subPath)
	if remotePath == "" {
		return filesystem.ErrUnauthorized
	}

	return s.withClient(ctx, func(client *minio.Client) error {
		_, err := client.PutObject(ctx, s.Config.Bucket, remotePath, data, -1,
			minio.PutObjectOptions{ContentType: "application/octet-stream"},
		)
		return err
	})
}

func (s S3) Delete(ctx context.Context, subPath string) error {
	remotePath := s.fullPath(subPath)
	if remotePath == "" {
		return filesystem.ErrUnauthorized
	}

	return s.withClient(ctx, func(client *minio.Client) error {
		// Check if it's a directory
		info, err := client.StatObject(ctx, s.Config.Bucket, dirKey(remotePath), minio.StatObjectOptions{})
		if err == nil && info.Key != "" {
			// Directory - delete all objects with this prefix
			return s.deleteRecursive(ctx, client, dirKey(remotePath))
		}

		// File - simple delete
		return client.RemoveObject(ctx, s.Config.Bucket, remotePath, minio.RemoveObjectOptions{})
	})
}

func (s S3) deleteRecursive(ctx context.Context, client *minio.Client, prefix string) error {
	opts := minio.ListObjectsOptions{Prefix: prefix, MaxKeys: s3ListPageSize}
	objectsCh := make(chan minio.ObjectInfo, s3ListPageSize)

	go func() {
		defer close(objectsCh)
		for obj := range client.ListObjects(ctx, s.Config.Bucket, opts) {
			if obj.Err != nil {
				return
			}
			objectsCh <- obj
		}
	}()

	for err := range client.RemoveObjects(ctx, s.Config.Bucket, objectsCh, minio.RemoveObjectsOptions{}) {
		if err.Err != nil {
			return err.Err
		}
	}

	return nil
}

func (s S3) Rename(ctx context.Context, oldSubPath, newSubPath string) error {
	oldPath := s.fullPath(oldSubPath)
	newPath := s.fullPath(newSubPath)
	if oldPath == "" || newPath == "" {
		return filesystem.ErrUnauthorized
	}

	return s.withClient(ctx, func(client *minio.Client) error {
		// Check if it's a directory
		info, err := client.StatObject(ctx, s.Config.Bucket, dirKey(oldPath), minio.StatObjectOptions{})
		if err == nil && info.Key != "" {
			return s.renameRecursive(ctx, client, dirKey(oldPath), dirKey(newPath))
		}

		// File rename = copy + delete
		src := minio.CopySrcOptions{
			Bucket: s.Config.Bucket,
			Object: oldPath,
		}
		dst := minio.CopyDestOptions{
			Bucket: s.Config.Bucket,
			Object: newPath,
		}
		if _, err := client.CopyObject(ctx, dst, src); err != nil {
			return err
		}
		return client.RemoveObject(ctx, s.Config.Bucket, oldPath, minio.RemoveObjectOptions{})
	})
}

func (s S3) renameRecursive(ctx context.Context, client *minio.Client, oldPrefix, newPrefix string) error {
	opts := minio.ListObjectsOptions{Prefix: oldPrefix, MaxKeys: s3ListPageSize}
	for obj := range client.ListObjects(ctx, s.Config.Bucket, opts) {
		if obj.Err != nil {
			return obj.Err
		}
		newKey := newPrefix + strings.TrimPrefix(obj.Key, oldPrefix)
		src := minio.CopySrcOptions{Bucket: s.Config.Bucket, Object: obj.Key}
		dst := minio.CopyDestOptions{Bucket: s.Config.Bucket, Object: newKey}
		if _, err := client.CopyObject(ctx, dst, src); err != nil {
			return err
		}
		if err := client.RemoveObject(ctx, s.Config.Bucket, obj.Key, minio.RemoveObjectOptions{}); err != nil {
			return err
		}
	}
	return nil
}

func (s S3) Mkdir(ctx context.Context, subPath string) error {
	remotePath := s.fullPath(subPath)
	if remotePath == "" {
		return filesystem.ErrUnauthorized
	}

	return s.withClient(ctx, func(client *minio.Client) error {
		_, err := client.PutObject(ctx, s.Config.Bucket, dirKey(remotePath),
			bytes.NewReader(nil), 0, minio.PutObjectOptions{ContentType: "application/x-directory"},
		)
		return err
	})
}

func (s S3) Touch(ctx context.Context, subPath string) error {
	remotePath := s.fullPath(subPath)
	if remotePath == "" {
		return filesystem.ErrUnauthorized
	}

	return s.withClient(ctx, func(client *minio.Client) error {
		_, err := client.StatObject(ctx, s.Config.Bucket, remotePath, minio.StatObjectOptions{})
		if err == nil {
			// File exists - update metadata by re-putting
			_, err = client.PutObject(ctx, s.Config.Bucket, remotePath,
				bytes.NewReader(nil), 0, minio.PutObjectOptions{ContentType: "application/octet-stream"},
			)
			return err
		}

		// File doesn't exist - create empty
		var s3Err minio.ErrorResponse
		if errors.As(err, &s3Err) && s3Err.Code == "NoSuchKey" {
			_, err = client.PutObject(ctx, s.Config.Bucket, remotePath,
				bytes.NewReader(nil), 0, minio.PutObjectOptions{ContentType: "application/octet-stream"},
			)
			return err
		}
		return err
	})
}

// sanitizedS3Root returns a human-readable root identifier for S3 sessions.
func sanitizedS3Root(cfg sessions.S3Config) string {
	endpoint := cfg.Endpoint
	scheme := "https"
	if !cfg.UseSSL {
		scheme = "http"
	}
	return fmt.Sprintf("s3://%s/%s/%s", scheme+"://"+endpoint, cfg.Bucket, strings.TrimPrefix(cfg.BasePath, "/"))
}
