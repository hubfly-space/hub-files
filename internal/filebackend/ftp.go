package filebackend

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"hubfly-files/internal/filesystem"
	"hubfly-files/internal/sessions"
	"io"
	"net"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	ftpclient "github.com/jlaffaye/ftp"
)

const (
	defaultFTPPort              = 21
	defaultFTPTimeout           = 15 * time.Second
	defaultFTPMaxConnPerSession = 4
)

type FTPPool struct {
	mu       sync.Mutex
	sessions map[string]*ftpSessionPool
}

type ftpSessionPool struct {
	entries []*ftpEntry
}

type ftpEntry struct {
	mu      sync.Mutex
	conn    *ftpclient.ServerConn
	lastUse time.Time
}

type FTP struct {
	Pool   *FTPPool
	Key    string
	Config sessions.FTPConfig
}

func NewFTPPool() *FTPPool {
	return &FTPPool{sessions: make(map[string]*ftpSessionPool)}
}

func (p *FTPPool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	var err error
	for key, sessionPool := range p.sessions {
		for _, entry := range sessionPool.entries {
			entry.mu.Lock()
			if entry.conn != nil {
				if closeErr := entry.conn.Quit(); closeErr != nil && err == nil {
					err = closeErr
				}
			}
			entry.mu.Unlock()
		}
		delete(p.sessions, key)
	}
	return err
}

func (p *FTPPool) Invalidate(key string) {
	p.mu.Lock()
	sessionPool := p.sessions[key]
	delete(p.sessions, key)
	p.mu.Unlock()

	if sessionPool == nil {
		return
	}
	for _, entry := range sessionPool.entries {
		entry.mu.Lock()
		if entry.conn != nil {
			_ = entry.conn.Quit()
		}
		entry.mu.Unlock()
	}
}

func (p *FTPPool) acquire(ctx context.Context, key string, cfg sessions.FTPConfig) (*ftpEntry, error) {
	if cfg.Port == 0 {
		cfg.Port = defaultFTPPort
	}

	for {
		p.mu.Lock()
		sessionPool := p.sessions[key]
		if sessionPool == nil {
			sessionPool = &ftpSessionPool{}
			p.sessions[key] = sessionPool
		}
		for _, entry := range sessionPool.entries {
			if entry.mu.TryLock() {
				entry.lastUse = time.Now()
				p.mu.Unlock()
				return entry, nil
			}
		}
		if len(sessionPool.entries) < defaultFTPMaxConnPerSession {
			entry := &ftpEntry{lastUse: time.Now()}
			entry.mu.Lock()
			sessionPool.entries = append(sessionPool.entries, entry)
			p.mu.Unlock()

			conn, err := dialFTP(ctx, cfg)
			if err != nil {
				entry.mu.Unlock()
				p.removeEntry(key, entry)
				return nil, err
			}
			entry.conn = conn
			return entry, nil
		}
		p.mu.Unlock()

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(50 * time.Millisecond):
		}
	}
}

func (p *FTPPool) removeEntry(key string, target *ftpEntry) {
	p.mu.Lock()
	defer p.mu.Unlock()
	sessionPool := p.sessions[key]
	if sessionPool == nil {
		return
	}
	for i, entry := range sessionPool.entries {
		if entry == target {
			sessionPool.entries = append(sessionPool.entries[:i], sessionPool.entries[i+1:]...)
			break
		}
	}
	if len(sessionPool.entries) == 0 {
		delete(p.sessions, key)
	}
}

func dialFTP(ctx context.Context, cfg sessions.FTPConfig) (*ftpclient.ServerConn, error) {
	dialCtx, cancel := context.WithTimeout(ctx, defaultFTPTimeout)
	defer cancel()

	address := net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port))
	conn, err := ftpclient.Dial(address, ftpclient.DialWithContext(dialCtx), ftpclient.DialWithTimeout(defaultFTPTimeout))
	if err != nil {
		return nil, err
	}
	if err := conn.Login(cfg.Username, cfg.Password); err != nil {
		_ = conn.Quit()
		return nil, err
	}
	if err := conn.Type(ftpclient.TransferTypeBinary); err != nil {
		_ = conn.Quit()
		return nil, err
	}
	return conn, nil
}

func (f FTP) withConnOnce(ctx context.Context, fn func(*ftpclient.ServerConn) error) error {
	entry, err := f.Pool.acquire(ctx, f.Key, f.Config)
	if err != nil {
		return err
	}
	defer entry.mu.Unlock()
	entry.lastUse = time.Now()
	return fn(entry.conn)
}

func (f FTP) withConn(ctx context.Context, fn func(*ftpclient.ServerConn) error) error {
	if err := f.withConnOnce(ctx, fn); err == nil {
		return nil
	} else {
		f.Pool.Invalidate(f.Key)
		retryErr := f.withConnOnce(ctx, fn)
		if retryErr != nil {
			return err
		}
		return nil
	}
}

func (f FTP) fullPath(subPath string) (string, error) {
	rel, err := safeFTPRelativePath(subPath)
	if err != nil {
		return "", err
	}
	base, err := safeFTPRelativePath(f.Config.BasePath)
	if err != nil {
		return "", err
	}

	joined := path.Join(base, rel)
	if joined == "." || joined == "/" {
		return ".", nil
	}
	return "/" + strings.TrimPrefix(joined, "/"), nil
}

func safeFTPRelativePath(input string) (string, error) {
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

func (f FTP) List(ctx context.Context, subPath string) ([]filesystem.FileInfo, error) {
	remotePath, err := f.fullPath(subPath)
	if err != nil {
		return nil, err
	}

	var files []filesystem.FileInfo
	err = f.withConn(ctx, func(conn *ftpclient.ServerConn) error {
		entries, err := conn.List(remotePath)
		if err != nil {
			return err
		}
		files = make([]filesystem.FileInfo, 0, len(entries))
		for _, entry := range entries {
			files = append(files, filesystem.FileInfo{
				Name:    entry.Name,
				IsDir:   entry.Type == ftpclient.EntryTypeFolder,
				Size:    int64(entry.Size),
				ModTime: entry.Time.Format("2006-01-02 15:04:05"),
			})
		}
		return nil
	})
	return files, err
}

func (f FTP) Storage(ctx context.Context, subPath string) (*filesystem.StorageInfo, error) {
	remotePath, err := f.fullPath(subPath)
	if err != nil {
		return nil, err
	}
	return &filesystem.StorageInfo{Path: remotePath}, nil
}

func (f FTP) Read(ctx context.Context, subPath string) (io.ReadCloser, error) {
	remotePath, err := f.fullPath(subPath)
	if err != nil {
		return nil, err
	}
	return f.read(ctx, remotePath, true)
}

func (f FTP) read(ctx context.Context, remotePath string, retry bool) (io.ReadCloser, error) {
	entry, err := f.Pool.acquire(ctx, f.Key, f.Config)
	if err != nil {
		return nil, err
	}
	entry.lastUse = time.Now()

	resp, err := entry.conn.Retr(remotePath)
	if err != nil {
		entry.mu.Unlock()
		if retry {
			f.Pool.Invalidate(f.Key)
			return f.read(ctx, remotePath, false)
		}
		return nil, err
	}

	return &ftpReadCloser{Response: resp, unlock: entry.mu.Unlock}, nil
}

type ftpReadCloser struct {
	*ftpclient.Response
	unlock func()
}

func (r *ftpReadCloser) Close() error {
	err := r.Response.Close()
	r.unlock()
	return err
}

func (f FTP) Write(ctx context.Context, subPath string, data io.Reader) error {
	return f.WriteAtomic(ctx, subPath, data)
}

func (f FTP) WriteAtomic(ctx context.Context, subPath string, data io.Reader) error {
	remotePath, err := f.fullPath(subPath)
	if err != nil {
		return err
	}

	return f.withConnOnce(ctx, func(conn *ftpclient.ServerConn) error {
		tmpPath, err := tempFTPPath(remotePath)
		if err != nil {
			return err
		}
		keepTemp := false
		defer func() {
			if !keepTemp {
				_ = conn.Delete(tmpPath)
			}
		}()

		if err := conn.Stor(tmpPath, data); err != nil {
			return err
		}
		if err := conn.Rename(tmpPath, remotePath); err != nil {
			_ = conn.Delete(remotePath)
			if retryErr := conn.Rename(tmpPath, remotePath); retryErr != nil {
				return err
			}
		}
		keepTemp = true
		return nil
	})
}

func tempFTPPath(remotePath string) (string, error) {
	random := make([]byte, 8)
	if _, err := rand.Read(random); err != nil {
		return "", err
	}
	dir, base := path.Split(remotePath)
	return path.Join(dir, "."+base+"."+hex.EncodeToString(random)+".upload"), nil
}

func (f FTP) Delete(ctx context.Context, subPath string) error {
	remotePath, err := f.fullPath(subPath)
	if err != nil {
		return err
	}

	return f.withConnOnce(ctx, func(conn *ftpclient.ServerConn) error {
		entry, err := conn.GetEntry(remotePath)
		if err == nil && entry.Type == ftpclient.EntryTypeFolder {
			return conn.RemoveDirRecur(remotePath)
		}
		if err := conn.Delete(remotePath); err == nil {
			return nil
		}
		return conn.RemoveDirRecur(remotePath)
	})
}

func (f FTP) Rename(ctx context.Context, oldSubPath, newSubPath string) error {
	oldRemotePath, err := f.fullPath(oldSubPath)
	if err != nil {
		return err
	}
	newRemotePath, err := f.fullPath(newSubPath)
	if err != nil {
		return err
	}
	return f.withConnOnce(ctx, func(conn *ftpclient.ServerConn) error {
		return conn.Rename(oldRemotePath, newRemotePath)
	})
}

func (f FTP) Mkdir(ctx context.Context, subPath string) error {
	remotePath, err := f.fullPath(subPath)
	if err != nil {
		return err
	}
	return f.withConnOnce(ctx, func(conn *ftpclient.ServerConn) error {
		return conn.MakeDir(remotePath)
	})
}

func (f FTP) Touch(ctx context.Context, subPath string) error {
	remotePath, err := f.fullPath(subPath)
	if err != nil {
		return err
	}
	return f.withConnOnce(ctx, func(conn *ftpclient.ServerConn) error {
		return conn.Stor(remotePath, bytes.NewReader(nil))
	})
}

func sanitizedFTPRoot(cfg sessions.FTPConfig) string {
	host := cfg.Host
	if cfg.Port != 0 && cfg.Port != defaultFTPPort {
		host = net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port))
	}
	return "ftp://" + host + "/" + strings.TrimPrefix(cfg.BasePath, "/")
}

func DefaultFTPCredentials(username, password string) (string, string) {
	if username == "" {
		username = "anonymous"
	}
	if username == "anonymous" && password == "" {
		password = "anonymous@"
	}
	return username, password
}
