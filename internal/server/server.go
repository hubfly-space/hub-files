package server

import (
	"encoding/json"
	"errors"
	"hubfly-files/internal/archive"
	"hubfly-files/internal/config"
	"hubfly-files/internal/filesystem"
	"hubfly-files/internal/sessions"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

const (
	// MaxRequestSize is the maximum allowed request body size (1 MB for JSON)
	MaxRequestSize = 1 << 20
	// Demo storage values exposed in demo mode.
	DemoTotalBytes = 20 << 20
	DemoUsedBytes  = 5 << 20
)

type Server struct {
	Config   *config.Config
	Sessions *sessions.Store
}

func NewServer(cfg *config.Config, store *sessions.Store) *Server {
	return &Server{
		Config:   cfg,
		Sessions: store,
	}
}

// maxBytesMiddleware limits the request body size
func maxBytesMiddleware(n int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, n)
			next.ServeHTTP(w, r)
		})
	}
}

func methodHandler(method string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != method {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		next(w, r)
	}
}

func methodHandlers(methods []string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		for _, method := range methods {
			if r.Method == method {
				next(w, r)
				return
			}
		}
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func writeFileSystemError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, filesystem.ErrUnauthorized):
		http.Error(w, "Invalid path", http.StatusBadRequest)
	case errors.Is(err, os.ErrNotExist):
		http.Error(w, "Not found", http.StatusNotFound)
	case errors.Is(err, os.ErrPermission):
		http.Error(w, "Permission denied", http.StatusForbidden)
	default:
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func writeUploadError(w http.ResponseWriter, err error) {
	var maxBytesErr *http.MaxBytesError
	switch {
	case errors.As(err, &maxBytesErr):
		http.Error(w, "File too large", http.StatusRequestEntityTooLarge)
	case errors.Is(err, filesystem.ErrUnauthorized):
		http.Error(w, "Invalid path", http.StatusBadRequest)
	case errors.Is(err, os.ErrNotExist):
		http.Error(w, "Not found", http.StatusNotFound)
	case errors.Is(err, os.ErrPermission):
		http.Error(w, "Permission denied", http.StatusForbidden)
	default:
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func writeArchiveError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, archive.ErrIllegalArchivePath), errors.Is(err, archive.ErrArchiveSymlink):
		http.Error(w, "Invalid archive", http.StatusBadRequest)
	case errors.Is(err, filesystem.ErrUnauthorized):
		http.Error(w, "Invalid path", http.StatusBadRequest)
	case errors.Is(err, os.ErrNotExist):
		http.Error(w, "Not found", http.StatusNotFound)
	case errors.Is(err, os.ErrPermission):
		http.Error(w, "Permission denied", http.StatusForbidden)
	default:
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func validUploadFilename(name string) bool {
	return name != "" &&
		name != "." &&
		name != ".." &&
		!filepath.IsAbs(name) &&
		!strings.ContainsAny(name, `/\`) &&
		!strings.ContainsRune(name, 0)
}

// Middleware to validate session
func (s *Server) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if strings.HasPrefix(token, "Bearer ") {
			token = strings.TrimPrefix(token, "Bearer ")
		}

		// Also check query parameter for compatibility (but log warning)
		if token == "" {
			token = r.URL.Query().Get("session")
			if token != "" {
				log.Printf("Warning: Using session token from URL query parameter - this is insecure")
			}
		}

		session, ok := s.Sessions.GetSession(token)
		if !ok {
			// Check for demo mode if no session
			if token == "" || token == "demo" {
				session = &sessions.Session{
					Root:        s.Config.DemoDir,
					ReadOnly:    true,
					AllowUpload: false,
					AllowEdit:   false,
					AllowDelete: false,
				}
				// Ensure demo dir exists
				os.MkdirAll(s.Config.DemoDir, 0755)
				r.Header.Set("X-Session-Demo", "true")
			} else {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
		}

		// Inject session info into request headers
		r.Header.Set("X-Session-Root", session.Root)
		if session.ReadOnly {
			r.Header.Set("X-Session-ReadOnly", "true")
		}
		// Inject permission flags
		if session.AllowUpload {
			r.Header.Set("X-Session-AllowUpload", "true")
		} else {
			r.Header.Set("X-Session-AllowUpload", "false")
		}
		if session.AllowEdit {
			r.Header.Set("X-Session-AllowEdit", "true")
		} else {
			r.Header.Set("X-Session-AllowEdit", "false")
		}
		if session.AllowDelete {
			r.Header.Set("X-Session-AllowDelete", "true")
		} else {
			r.Header.Set("X-Session-AllowDelete", "false")
		}

		next(w, r)
	}
}

func (s *Server) checkReadOnly(w http.ResponseWriter, r *http.Request) bool {
	if r.Header.Get("X-Session-ReadOnly") == "true" {
		http.Error(w, "Read-only session", http.StatusForbidden)
		return true
	}
	return false
}

func (s *Server) checkPermission(w http.ResponseWriter, r *http.Request, permFlag string) bool {
	if r.Header.Get("X-Session-ReadOnly") == "true" {
		http.Error(w, "Read-only session", http.StatusForbidden)
		return false
	}

	sessionPerm := r.Header.Get("X-Session-" + permFlag)
	if sessionPerm == "false" {
		http.Error(w, permFlag+" not allowed for this session", http.StatusForbidden)
		return false
	}
	return true
}

func (s *Server) handleList(w http.ResponseWriter, r *http.Request) {
	root := r.Header.Get("X-Session-Root")
	path := r.URL.Query().Get("path")

	files, err := filesystem.ListDir(root, path)
	if err != nil {
		log.Printf("List error: %v", err)
		writeFileSystemError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(files)
}

func (s *Server) handleStorage(w http.ResponseWriter, r *http.Request) {
	root := r.Header.Get("X-Session-Root")
	path := r.URL.Query().Get("path")

	if r.Header.Get("X-Session-Demo") == "true" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(&filesystem.StorageInfo{
			Path:           root,
			TotalBytes:     DemoTotalBytes,
			UsedBytes:      DemoUsedBytes,
			AvailableBytes: DemoTotalBytes - DemoUsedBytes,
			UsedPercent:    (float64(DemoUsedBytes) / float64(DemoTotalBytes)) * 100,
		})
		return
	}

	storage, err := filesystem.GetStorageInfo(root, path)
	if err != nil {
		log.Printf("Storage error: %v", err)
		writeFileSystemError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(storage)
}

func (s *Server) handleGetFile(w http.ResponseWriter, r *http.Request) {
	root := r.Header.Get("X-Session-Root")
	path := r.URL.Query().Get("path")

	reader, err := filesystem.ReadFile(root, path)
	if err != nil {
		log.Printf("GetFile error: %v", err)
		writeFileSystemError(w, err)
		return
	}
	defer reader.Close()

	if r.URL.Query().Get("download") == "1" {
		w.Header().Set("Content-Disposition", "attachment; filename=\""+filepath.Base(path)+"\"")
		w.Header().Set("Content-Type", "application/octet-stream")
	}

	io.Copy(w, reader)
}

func (s *Server) handlePutFile(w http.ResponseWriter, r *http.Request) {
	if !s.checkPermission(w, r, "AllowEdit") {
		return
	}
	root := r.Header.Get("X-Session-Root")
	path := r.URL.Query().Get("path")

	err := filesystem.WriteFile(root, path, r.Body)
	if err != nil {
		log.Printf("PutFile error: %v", err)
		writeFileSystemError(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleUpload(w http.ResponseWriter, r *http.Request) {
	if !s.checkPermission(w, r, "AllowUpload") {
		return
	}

	root := r.Header.Get("X-Session-Root")
	maxUploadBytes := s.Config.MaxUploadBytes
	if maxUploadBytes > 0 {
		if r.ContentLength > maxUploadBytes {
			http.Error(w, "File too large", http.StatusRequestEntityTooLarge)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes)
	}

	if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
		s.handleMultipartUpload(w, r, root)
		return
	}

	dirPath := r.URL.Query().Get("path")
	filename := r.URL.Query().Get("filename")
	if !validUploadFilename(filename) {
		http.Error(w, "Invalid file name", http.StatusBadRequest)
		return
	}

	finalPath := filepath.Join(dirPath, filename)
	if err := filesystem.WriteFileAtomic(root, finalPath, r.Body); err != nil {
		log.Printf("Upload error: %v", err)
		writeUploadError(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleMultipartUpload(w http.ResponseWriter, r *http.Request, root string) {
	reader, err := r.MultipartReader()
	if err != nil {
		http.Error(w, "Invalid file upload", http.StatusBadRequest)
		return
	}

	dirPath := r.URL.Query().Get("path")
	for {
		part, err := reader.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			writeUploadError(w, err)
			return
		}

		switch part.FormName() {
		case "path":
			if dirPath == "" {
				pathBytes, err := io.ReadAll(io.LimitReader(part, MaxRequestSize+1))
				if err != nil {
					writeUploadError(w, err)
					return
				}
				if len(pathBytes) > MaxRequestSize {
					http.Error(w, "Invalid path", http.StatusBadRequest)
					return
				}
				dirPath = string(pathBytes)
			}
		case "file":
			filename := part.FileName()
			if !validUploadFilename(filename) {
				http.Error(w, "Invalid file name", http.StatusBadRequest)
				return
			}

			finalPath := filepath.Join(dirPath, filename)
			if err := filesystem.WriteFileAtomic(root, finalPath, part); err != nil {
				log.Printf("Upload error: %v", err)
				writeUploadError(w, err)
				return
			}
			w.WriteHeader(http.StatusOK)
			return
		}
	}

	http.Error(w, "Invalid file upload", http.StatusBadRequest)
}

func (s *Server) handleMkdir(w http.ResponseWriter, r *http.Request) {
	if s.checkReadOnly(w, r) {
		return
	}
	root := r.Header.Get("X-Session-Root")
	var req struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	err := filesystem.Mkdir(root, req.Path)
	if err != nil {
		log.Printf("Mkdir error: %v", err)
		writeFileSystemError(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleRename(w http.ResponseWriter, r *http.Request) {
	if s.checkReadOnly(w, r) {
		return
	}
	root := r.Header.Get("X-Session-Root")
	var req struct {
		OldPath string `json:"oldPath"`
		NewPath string `json:"newPath"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	err := filesystem.Rename(root, req.OldPath, req.NewPath)
	if err != nil {
		log.Printf("Rename error: %v", err)
		writeFileSystemError(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleDelete(w http.ResponseWriter, r *http.Request) {
	if !s.checkPermission(w, r, "AllowDelete") {
		return
	}
	root := r.Header.Get("X-Session-Root")
	path := r.URL.Query().Get("path")

	err := filesystem.DeleteFile(root, path)
	if err != nil {
		log.Printf("Delete error: %v", err)
		writeFileSystemError(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleZip(w http.ResponseWriter, r *http.Request) {
	if s.checkReadOnly(w, r) {
		return
	}
	root := r.Header.Get("X-Session-Root")
	var req struct {
		Source string `json:"source"`
		Target string `json:"target"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	src, err := filesystem.SafePath(root, req.Source)
	if err != nil {
		http.Error(w, "Invalid source path", http.StatusBadRequest)
		return
	}
	dst, err := filesystem.SafePath(root, req.Target)
	if err != nil {
		http.Error(w, "Invalid target path", http.StatusBadRequest)
		return
	}

	ownership, err := filesystem.OwnershipForPath(root, req.Target)
	if err != nil {
		log.Printf("Zip ownership error: %v", err)
		writeFileSystemError(w, err)
		return
	}

	err = archive.Zip(src, dst, ownership)
	if err != nil {
		log.Printf("Zip error: %v", err)
		writeArchiveError(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleExtract(w http.ResponseWriter, r *http.Request) {
	if s.checkReadOnly(w, r) {
		return
	}
	root := r.Header.Get("X-Session-Root")
	var req struct {
		Source string `json:"source"`
		Target string `json:"target"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	src, err := filesystem.SafePath(root, req.Source)
	if err != nil {
		http.Error(w, "Invalid source path", http.StatusBadRequest)
		return
	}
	dst, err := filesystem.SafePath(root, req.Target)
	if err != nil {
		http.Error(w, "Invalid target path", http.StatusBadRequest)
		return
	}

	ownership, err := filesystem.OwnershipForPath(root, req.Target)
	if err != nil {
		log.Printf("Unzip ownership error: %v", err)
		writeFileSystemError(w, err)
		return
	}

	err = archive.Unzip(src, dst, ownership)
	if err != nil {
		log.Printf("Unzip error: %v", err)
		writeArchiveError(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleCreateSession(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Root        string `json:"root"`
		TTLSeconds  int    `json:"ttlSeconds"`
		ReadOnly    bool   `json:"readonly"`
		AllowUpload bool   `json:"allowUpload"`
		AllowEdit   bool   `json:"allowEdit"`
		AllowDelete bool   `json:"allowDelete"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate inputs
	if req.Root == "" {
		http.Error(w, "root is required", http.StatusBadRequest)
		return
	}
	if req.TTLSeconds <= 0 || req.TTLSeconds > 86400*30 { // Max 30 days
		http.Error(w, "ttlSeconds must be between 1 and 2592000", http.StatusBadRequest)
		return
	}

	// Clean and validate the root path
	cleanRoot := filepath.Clean(req.Root)
	absRoot, err := filepath.Abs(cleanRoot)
	if err != nil {
		http.Error(w, "Invalid root path", http.StatusBadRequest)
		return
	}

	session, err := s.Sessions.CreateSession(absRoot, req.TTLSeconds, req.ReadOnly, req.AllowUpload, req.AllowEdit, req.AllowDelete)
	if err != nil {
		log.Printf("CreateSession error: %v", err)
		switch {
		case errors.Is(err, sessions.ErrRateLimitExceeded):
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
		case errors.Is(err, sessions.ErrMaxSessions):
			http.Error(w, "Max sessions reached", http.StatusConflict)
		default:
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(session)
}

func (s *Server) handleUI(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/api/") {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	uiDir := s.Config.UIDir
	if uiDir == "" {
		http.Error(w, "UI directory not configured", http.StatusServiceUnavailable)
		return
	}

	cleanPath := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
	targetPath := filepath.Join(uiDir, filepath.FromSlash(cleanPath))
	indexPath := filepath.Join(uiDir, "index.html")

	if cleanPath == "" || cleanPath == "." {
		http.ServeFile(w, r, indexPath)
		return
	}

	info, err := os.Stat(targetPath)
	if err == nil && !info.IsDir() {
		http.ServeFile(w, r, targetPath)
		return
	}

	http.ServeFile(w, r, indexPath)
}

func (s *Server) SetupRoutes() *http.ServeMux {
	mux := http.NewServeMux()

	// UI API
	mux.HandleFunc("/api/list", s.authMiddleware(methodHandler(http.MethodGet, s.handleList)))
	mux.HandleFunc("/api/storage", s.authMiddleware(methodHandler(http.MethodGet, s.handleStorage)))
	mux.HandleFunc("/api/file", s.authMiddleware(methodHandlers([]string{http.MethodGet, http.MethodPut}, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			s.handleGetFile(w, r)
			return
		}
		s.handlePutFile(w, r)
	})))
	mux.HandleFunc("/api/upload", s.authMiddleware(methodHandler(http.MethodPost, s.handleUpload)))
	// JSON endpoints with max bytes middleware
	mux.Handle("/api/mkdir", maxBytesMiddleware(MaxRequestSize)(s.authMiddleware(methodHandler(http.MethodPost, s.handleMkdir))))
	mux.Handle("/api/rename", maxBytesMiddleware(MaxRequestSize)(s.authMiddleware(methodHandler(http.MethodPost, s.handleRename))))
	mux.Handle("/api/delete", maxBytesMiddleware(MaxRequestSize)(s.authMiddleware(methodHandler(http.MethodDelete, s.handleDelete))))
	mux.Handle("/api/zip", maxBytesMiddleware(MaxRequestSize)(s.authMiddleware(methodHandler(http.MethodPost, s.handleZip))))
	mux.Handle("/api/extract", maxBytesMiddleware(MaxRequestSize)(s.authMiddleware(methodHandler(http.MethodPost, s.handleExtract))))
	mux.HandleFunc("/", s.handleUI)

	return mux
}

func (s *Server) SetupManagementRoutes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/sessions", methodHandler(http.MethodPost, s.handleCreateSession))
	return mux
}
