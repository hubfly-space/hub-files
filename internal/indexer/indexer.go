package indexer

import (
	"errors"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"unsafe"

	"golang.org/x/sys/unix"

	"hubfly-files/internal/sqlite"
)

const watchMask = unix.IN_CREATE |
	unix.IN_DELETE |
	unix.IN_MOVED_FROM |
	unix.IN_MOVED_TO |
	unix.IN_CLOSE_WRITE |
	unix.IN_ATTRIB |
	unix.IN_DELETE_SELF |
	unix.IN_MOVE_SELF

// excludedDirs lists directory names that should not be indexed or watched.
// These are typically large dependency/build/cache directories that add noise to search results.
var excludedDirs = map[string]bool{
	"node_modules":    true,
	".git":            true,
	".svn":            true,
	".hg":             true,
	".next":           true,
	".nuxt":           true,
	".cache":          true,
	".turbo":          true,
	"bower_components": true,
	"target":          true,
	"vendor":          true,
	"__pycache__":     true,
	".venv":           true,
	".idea":           true,
}

func isExcludedDir(name string) bool {
	return excludedDirs[name]
}

type Manager struct {
	store *sqlite.Storage

	mu    sync.Mutex
	roots map[string]*RootIndexer
}

func NewManager(store *sqlite.Storage) *Manager {
	return &Manager{
		store: store,
		roots: make(map[string]*RootIndexer),
	}
}

func (m *Manager) EnsureRoot(root string) error {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	absRoot = filepath.Clean(absRoot)

	info, err := os.Stat(absRoot)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return errors.New("root is not a directory")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.roots[absRoot]; ok {
		return nil
	}

	rootIndexer, err := newRootIndexer(absRoot, m.store)
	if err != nil {
		return err
	}
	m.roots[absRoot] = rootIndexer
	rootIndexer.start()

	return nil
}

func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for root, indexer := range m.roots {
		if err := indexer.Close(); err != nil {
			log.Printf("indexer close error for %s: %v", root, err)
		}
		delete(m.roots, root)
	}

	return nil
}

type RootIndexer struct {
	root  string
	store *sqlite.Storage
	fd    int

	mu      sync.Mutex
	wdToDir map[int]string
	dirToWD map[string]int

	stop chan struct{}
	once sync.Once
}

func newRootIndexer(root string, store *sqlite.Storage) (*RootIndexer, error) {
	fd, err := unix.InotifyInit1(unix.IN_CLOEXEC)
	if err != nil {
		return nil, err
	}

	return &RootIndexer{
		root:    root,
		store:   store,
		fd:      fd,
		wdToDir: make(map[int]string),
		dirToWD: make(map[string]int),
		stop:    make(chan struct{}),
	}, nil
}

func (r *RootIndexer) start() {
	go r.readEvents()
	go func() {
		if err := r.indexTree(r.root); err != nil {
			log.Printf("initial index error for %s: %v", r.root, err)
		}
	}()
}

func (r *RootIndexer) Close() error {
	var err error
	r.once.Do(func() {
		close(r.stop)
		r.mu.Lock()
		for _, wd := range r.dirToWD {
			_, _ = unix.InotifyRmWatch(r.fd, uint32(wd))
		}
		r.wdToDir = make(map[int]string)
		r.dirToWD = make(map[string]int)
		r.mu.Unlock()
		err = unix.Close(r.fd)
	})
	return err
}

func (r *RootIndexer) indexTree(start string) error {
	return filepath.WalkDir(start, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			if path == start {
				return walkErr
			}
			log.Printf("index walk skipped %s: %v", path, walkErr)
			return nil
		}

		info, err := entry.Info()
		if err != nil {
			log.Printf("index stat skipped %s: %v", path, err)
			return nil
		}

		if info.IsDir() && info.Mode()&os.ModeSymlink == 0 {
			if isExcludedDir(entry.Name()) {
				if relPath, ok := r.relPath(path); ok {
					if err := r.store.DeletePathPrefix(r.root, relPath); err != nil {
						log.Printf("index cleanup error for %s: %v", path, err)
					}
				}
				return filepath.SkipDir
			}
			if err := r.addWatch(path); err != nil {
				log.Printf("watch error for %s: %v", path, err)
			}
		}

		if filepath.Clean(path) == r.root {
			return nil
		}

		return r.indexInfo(path, info)
	})
}

func (r *RootIndexer) indexInfo(path string, info os.FileInfo) error {
	relPath, ok := r.relPath(path)
	if !ok || relPath == "" {
		return nil
	}

	parentPath := filepath.ToSlash(filepath.Dir(relPath))
	if parentPath == "." {
		parentPath = ""
	}

	extension := ""
	if !info.IsDir() {
		extension = filepath.Ext(info.Name())
	}

	return r.store.UpsertFileEntry(&sqlite.FileEntries{
		Root:       r.root,
		RelPath:    relPath,
		BaseName:   info.Name(),
		ParentPath: parentPath,
		IsDir:      info.IsDir(),
		Size:       info.Size(),
		ModTime:    info.ModTime(),
		Extension:  extension,
	})
}

func (r *RootIndexer) addWatch(path string) error {
	path = filepath.Clean(path)

	r.mu.Lock()
	if _, ok := r.dirToWD[path]; ok {
		r.mu.Unlock()
		return nil
	}
	r.mu.Unlock()

	wd, err := unix.InotifyAddWatch(r.fd, path, watchMask)
	if err != nil {
		return err
	}

	r.mu.Lock()
	r.wdToDir[wd] = path
	r.dirToWD[path] = wd
	r.mu.Unlock()

	return nil
}

func (r *RootIndexer) removeWatchPrefix(path string) {
	path = filepath.Clean(path)
	sepPath := path + string(os.PathSeparator)

	r.mu.Lock()
	defer r.mu.Unlock()

	for watched, wd := range r.dirToWD {
		if watched == path || strings.HasPrefix(watched, sepPath) {
			_, _ = unix.InotifyRmWatch(r.fd, uint32(wd))
			delete(r.dirToWD, watched)
			delete(r.wdToDir, wd)
		}
	}
}

func (r *RootIndexer) removeWatchByWD(wd int) string {
	r.mu.Lock()
	defer r.mu.Unlock()

	dir := r.wdToDir[wd]
	if dir != "" {
		delete(r.wdToDir, wd)
		delete(r.dirToWD, dir)
	}

	return dir
}

func (r *RootIndexer) dirForWD(wd int) string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.wdToDir[wd]
}

func (r *RootIndexer) readEvents() {
	buf := make([]byte, unix.SizeofInotifyEvent*64)

	for {
		select {
		case <-r.stop:
			return
		default:
		}

		n, err := unix.Read(r.fd, buf)
		if err != nil {
			select {
			case <-r.stop:
				return
			default:
			}
			if errors.Is(err, unix.EINTR) {
				continue
			}
			if errors.Is(err, unix.EBADF) {
				return
			}
			log.Printf("inotify read error for %s: %v", r.root, err)
			return
		}

		for offset := 0; offset+unix.SizeofInotifyEvent <= n; {
			event := (*unix.InotifyEvent)(unsafe.Pointer(&buf[offset]))
			offset += unix.SizeofInotifyEvent

			name := ""
			if event.Len > 0 {
				nameBytes := buf[offset : offset+int(event.Len)]
				name = strings.TrimRight(string(nameBytes), "\x00")
			}
			offset += int(event.Len)

			r.handleEvent(int(event.Wd), event.Mask, name)
		}
	}
}

func (r *RootIndexer) handleEvent(wd int, mask uint32, name string) {
	if mask&unix.IN_Q_OVERFLOW != 0 {
		log.Printf("inotify queue overflow for %s; rebuilding index", r.root)
		if err := r.indexTree(r.root); err != nil {
			log.Printf("index rebuild error for %s: %v", r.root, err)
		}
		return
	}

	dir := r.dirForWD(wd)
	if dir == "" {
		return
	}

	if mask&unix.IN_IGNORED != 0 {
		r.removeWatchByWD(wd)
		return
	}

	path := dir
	if name != "" {
		path = filepath.Join(dir, name)
	}

	if mask&(unix.IN_DELETE_SELF|unix.IN_MOVE_SELF) != 0 {
		r.removeWatchPrefix(path)
		if relPath, ok := r.relPath(path); ok {
			if err := r.store.DeletePathPrefix(r.root, relPath); err != nil {
				log.Printf("index delete error for %s: %v", path, err)
			}
		}
		return
	}

	if name == "" {
		return
	}

	if mask&(unix.IN_DELETE|unix.IN_MOVED_FROM) != 0 {
		if mask&unix.IN_ISDIR != 0 {
			r.removeWatchPrefix(path)
		}
		if relPath, ok := r.relPath(path); ok {
			if err := r.store.DeletePathPrefix(r.root, relPath); err != nil {
				log.Printf("index delete error for %s: %v", path, err)
			}
		}
		return
	}

	if mask&(unix.IN_CREATE|unix.IN_MOVED_TO) != 0 {
		info, err := os.Lstat(path)
		if err != nil {
			return
		}
		if info.IsDir() && info.Mode()&os.ModeSymlink == 0 {
			if isExcludedDir(info.Name()) {
				return
			}
			if err := r.indexTree(path); err != nil {
				log.Printf("index tree error for %s: %v", path, err)
			}
			return
		}
		if err := r.indexInfo(path, info); err != nil {
			log.Printf("index update error for %s: %v", path, err)
		}
		return
	}

	if mask&(unix.IN_CLOSE_WRITE|unix.IN_ATTRIB) != 0 {
		info, err := os.Lstat(path)
		if err != nil {
			return
		}
		if err := r.indexInfo(path, info); err != nil {
			log.Printf("index update error for %s: %v", path, err)
		}
	}
}

func (r *RootIndexer) relPath(path string) (string, bool) {
	rel, err := filepath.Rel(r.root, filepath.Clean(path))
	if err != nil {
		return "", false
	}
	if rel == "." {
		return "", true
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", false
	}
	return filepath.ToSlash(rel), true
}
