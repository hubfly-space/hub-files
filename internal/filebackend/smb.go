package filebackend

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"hubfly-files/internal/filesystem"
	"hubfly-files/internal/sessions"
	"io"
	"net"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/hirochachacha/go-smb2"
)

const (
	defaultSMBPort     = 445
	defaultDialTimeout = 10 * time.Second
)

type SMBPool struct {
	mu      sync.Mutex
	entries map[string]*smbEntry
}

type smbEntry struct {
	conn    net.Conn
	session *smb2.Session
	share   *smb2.Share
	lastUse time.Time
}

type SMB struct {
	Pool   *SMBPool
	Key    string
	Config sessions.SMBConfig
}

func NewSMBPool() *SMBPool {
	return &SMBPool{entries: make(map[string]*smbEntry)}
}

func (p *SMBPool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	var err error
	for key, entry := range p.entries {
		if closeErr := closeSMBEntry(entry); closeErr != nil && err == nil {
			err = closeErr
		}
		delete(p.entries, key)
	}
	return err
}

func (p *SMBPool) Invalidate(key string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	entry := p.entries[key]
	if entry != nil {
		_ = closeSMBEntry(entry)
		delete(p.entries, key)
	}
}

func (p *SMBPool) mount(ctx context.Context, key string, cfg sessions.SMBConfig) (*smb2.Share, error) {
	if cfg.Port == 0 {
		cfg.Port = defaultSMBPort
	}

	p.mu.Lock()
	if entry := p.entries[key]; entry != nil {
		entry.lastUse = time.Now()
		share := entry.share
		p.mu.Unlock()
		return share, nil
	}
	p.mu.Unlock()

	dialCtx, cancel := context.WithTimeout(ctx, defaultDialTimeout)
	defer cancel()

	address := net.JoinHostPort(cfg.Host, fmt.Sprintf("%d", cfg.Port))
	conn, err := (&net.Dialer{Timeout: defaultDialTimeout, KeepAlive: 30 * time.Second}).DialContext(dialCtx, "tcp", address)
	if err != nil {
		return nil, err
	}

	dialer := &smb2.Dialer{
		MaxCreditBalance: 128,
		Initiator: &smb2.NTLMInitiator{
			User:        cfg.Username,
			Password:    cfg.Password,
			Domain:      cfg.Domain,
			Workstation: cfg.Workstation,
		},
	}

	session, err := dialer.DialContext(dialCtx, conn)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}

	share, err := session.Mount(cfg.Share)
	if err != nil {
		_ = session.Logoff()
		_ = conn.Close()
		return nil, err
	}

	entry := &smbEntry{conn: conn, session: session, share: share, lastUse: time.Now()}

	p.mu.Lock()
	if existing := p.entries[key]; existing != nil {
		p.mu.Unlock()
		_ = closeSMBEntry(entry)
		return existing.share, nil
	}
	p.entries[key] = entry
	p.mu.Unlock()

	return share, nil
}

func closeSMBEntry(entry *smbEntry) error {
	var err error
	if entry.share != nil {
		if closeErr := entry.share.Umount(); closeErr != nil && err == nil {
			err = closeErr
		}
	}
	if entry.session != nil {
		if closeErr := entry.session.Logoff(); closeErr != nil && err == nil {
			err = closeErr
		}
	}
	if entry.conn != nil {
		if closeErr := entry.conn.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}
	return err
}

func (s SMB) withShareOnce(ctx context.Context, fn func(*smb2.Share) error) error {
	share, err := s.Pool.mount(ctx, s.Key, s.Config)
	if err != nil {
		return err
	}
	return fn(share)
}

func (s SMB) withShare(ctx context.Context, fn func(*smb2.Share) error) error {
	if err := s.withShareOnce(ctx, fn); err == nil {
		return nil
	} else {
		// One reconnect keeps stale TCP sessions from making browsing feel flaky.
		s.Pool.Invalidate(s.Key)
		retryErr := s.withShareOnce(ctx, fn)
		if retryErr != nil {
			return err
		}
		return nil
	}
}

func (s SMB) fullPath(subPath string) (string, error) {
	rel, err := safeSMBRelativePath(subPath)
	if err != nil {
		return "", err
	}
	base, err := safeSMBRelativePath(s.Config.BasePath)
	if err != nil {
		return "", err
	}

	joined := path.Join(base, rel)
	if joined == "." || joined == "/" {
		return ".", nil
	}
	return joined, nil
}

func safeSMBRelativePath(input string) (string, error) {
	input = strings.ReplaceAll(input, "\\", "/")
	if strings.ContainsRune(input, 0) {
		return "", filesystem.ErrUnauthorized
	}

	parts := strings.Split(input, "/")
	clean := make([]string, 0, len(parts))
	for _, part := range parts {
		switch part {
		case "", ".":
			continue
		case "..":
			return "", filesystem.ErrUnauthorized
		default:
			clean = append(clean, part)
		}
	}

	return strings.Join(clean, "/"), nil
}

func (s SMB) List(ctx context.Context, subPath string) ([]filesystem.FileInfo, error) {
	remotePath, err := s.fullPath(subPath)
	if err != nil {
		return nil, err
	}

	var files []filesystem.FileInfo
	err = s.withShare(ctx, func(share *smb2.Share) error {
		entries, err := share.ReadDir(remotePath)
		if err != nil {
			return err
		}
		files = make([]filesystem.FileInfo, 0, len(entries))
		for _, entry := range entries {
			files = append(files, filesystem.FileInfo{
				Name:    entry.Name(),
				IsDir:   entry.IsDir(),
				Size:    entry.Size(),
				ModTime: entry.ModTime().Format("2006-01-02 15:04:05"),
			})
		}
		return nil
	})
	return files, err
}

func (s SMB) Storage(ctx context.Context, subPath string) (*filesystem.StorageInfo, error) {
	remotePath, err := s.fullPath(subPath)
	if err != nil {
		return nil, err
	}

	var storage *filesystem.StorageInfo
	err = s.withShare(ctx, func(share *smb2.Share) error {
		stats, err := share.Statfs(remotePath)
		if err != nil {
			return err
		}
		blockSize := stats.BlockSize()
		total := stats.TotalBlockCount() * blockSize
		available := stats.AvailableBlockCount() * blockSize
		used := (stats.TotalBlockCount() - stats.FreeBlockCount()) * blockSize
		usedPercent := 0.0
		if total > 0 {
			usedPercent = (float64(used) / float64(total)) * 100
		}
		storage = &filesystem.StorageInfo{
			Path:           s.Config.BasePath,
			TotalBytes:     total,
			UsedBytes:      used,
			AvailableBytes: available,
			UsedPercent:    usedPercent,
		}
		return nil
	})
	return storage, err
}

func (s SMB) Read(ctx context.Context, subPath string) (io.ReadCloser, error) {
	remotePath, err := s.fullPath(subPath)
	if err != nil {
		return nil, err
	}

	var file *smb2.File
	err = s.withShare(ctx, func(share *smb2.Share) error {
		opened, err := share.Open(remotePath)
		if err != nil {
			return err
		}
		file = opened
		return nil
	})
	if err != nil {
		return nil, err
	}
	return file, nil
}

func (s SMB) Write(ctx context.Context, subPath string, data io.Reader) error {
	remotePath, err := s.fullPath(subPath)
	if err != nil {
		return err
	}

	return s.withShareOnce(ctx, func(share *smb2.Share) error {
		file, err := share.OpenFile(remotePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(file, data)
		closeErr := file.Close()
		if copyErr != nil {
			return copyErr
		}
		return closeErr
	})
}

func (s SMB) WriteAtomic(ctx context.Context, subPath string, data io.Reader) error {
	remotePath, err := s.fullPath(subPath)
	if err != nil {
		return err
	}

	return s.withShareOnce(ctx, func(share *smb2.Share) error {
		tmpPath, err := tempRemotePath(remotePath)
		if err != nil {
			return err
		}

		file, err := share.OpenFile(tmpPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
		if err != nil {
			return err
		}

		keepTemp := false
		defer func() {
			if !keepTemp {
				_ = share.Remove(tmpPath)
			}
		}()

		_, copyErr := io.Copy(file, data)
		closeErr := file.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
		if err := share.Rename(tmpPath, remotePath); err != nil {
			return err
		}
		keepTemp = true
		return nil
	})
}

func tempRemotePath(remotePath string) (string, error) {
	random := make([]byte, 8)
	if _, err := rand.Read(random); err != nil {
		return "", err
	}
	dir, base := path.Split(remotePath)
	return path.Join(dir, "."+base+"."+hex.EncodeToString(random)+".upload"), nil
}

func (s SMB) Delete(ctx context.Context, subPath string) error {
	remotePath, err := s.fullPath(subPath)
	if err != nil {
		return err
	}
	return s.withShareOnce(ctx, func(share *smb2.Share) error {
		return share.RemoveAll(remotePath)
	})
}

func (s SMB) Rename(ctx context.Context, oldSubPath, newSubPath string) error {
	oldRemotePath, err := s.fullPath(oldSubPath)
	if err != nil {
		return err
	}
	newRemotePath, err := s.fullPath(newSubPath)
	if err != nil {
		return err
	}
	return s.withShareOnce(ctx, func(share *smb2.Share) error {
		return share.Rename(oldRemotePath, newRemotePath)
	})
}

func (s SMB) Mkdir(ctx context.Context, subPath string) error {
	remotePath, err := s.fullPath(subPath)
	if err != nil {
		return err
	}
	return s.withShareOnce(ctx, func(share *smb2.Share) error {
		return share.MkdirAll(remotePath, 0755)
	})
}

func (s SMB) Touch(ctx context.Context, subPath string) error {
	remotePath, err := s.fullPath(subPath)
	if err != nil {
		return err
	}
	return s.withShareOnce(ctx, func(share *smb2.Share) error {
		file, err := share.OpenFile(remotePath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
		if err == nil {
			return file.Close()
		}
		if errors.Is(err, os.ErrExist) {
			return nil
		}
		return err
	})
}
