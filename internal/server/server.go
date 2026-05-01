package server

import (
	"encoding/json"
	"hubfly-files/internal/archive"
	"hubfly-files/internal/config"
	"hubfly-files/internal/filesystem"
	"hubfly-files/internal/sessions"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const (
	// MaxUploadSize is the maximum allowed upload size (10 MB)
	MaxUploadSize = 10 << 20
	// MaxRequestSize is the maximum allowed request body size (1 MB for JSON)
	MaxRequestSize = 1 << 20
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
					Root:     s.Config.DemoDir,
					ReadOnly: true,
				}
				// Ensure demo dir exists
				os.MkdirAll(s.Config.DemoDir, 0755)
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

func (s *Server) handleList(w http.ResponseWriter, r *http.Request) {
	root := r.Header.Get("X-Session-Root")
	path := r.URL.Query().Get("path")

	files, err := filesystem.ListDir(root, path)
	if err != nil {
		log.Printf("List error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(files)
}

func (s *Server) handleGetFile(w http.ResponseWriter, r *http.Request) {
	root := r.Header.Get("X-Session-Root")
	path := r.URL.Query().Get("path")

	reader, err := filesystem.ReadFile(root, path)
	if err != nil {
		log.Printf("GetFile error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer reader.Close()

	io.Copy(w, reader)
}

func (s *Server) handlePutFile(w http.ResponseWriter, r *http.Request) {
	if s.checkReadOnly(w, r) {
		return
	}
	root := r.Header.Get("X-Session-Root")
	path := r.URL.Query().Get("path")

	err := filesystem.WriteFile(root, path, r.Body)
	if err != nil {
		log.Printf("PutFile error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleUpload(w http.ResponseWriter, r *http.Request) {
	if s.checkReadOnly(w, r) {
		return
	}

	// Limit request body size for uploads
	r.Body = http.MaxBytesReader(w, r.Body, MaxUploadSize)

	root := r.Header.Get("X-Session-Root")
	path := r.FormValue("path")

	// Parse multipart form with size limit
	if err := r.ParseMultipartForm(MaxUploadSize); err != nil {
		http.Error(w, "File too large (max 10MB)", http.StatusRequestEntityTooLarge)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Invalid file upload", http.StatusBadRequest)
		return
	}
	defer file.Close()

	finalPath := filepath.Join(path, header.Filename)
	err = filesystem.WriteFile(root, finalPath, file)
	if err != nil {
		log.Printf("Upload error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleMkdir(w http.ResponseWriter, r *http.Request) {
	if s.checkReadOnly(w, r) {
		return
	}
	root := r.Header.Get("X-Session-Root")
	var req struct{ Path string `json:"path"` }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	err := filesystem.Mkdir(root, req.Path)
	if err != nil {
		log.Printf("Mkdir error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
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
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleDelete(w http.ResponseWriter, r *http.Request) {
	if s.checkReadOnly(w, r) {
		return
	}
	root := r.Header.Get("X-Session-Root")
	path := r.URL.Query().Get("path")

	err := filesystem.DeleteFile(root, path)
	if err != nil {
		log.Printf("Delete error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
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

	err = archive.Zip(src, dst)
	if err != nil {
		log.Printf("Zip error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
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

	err = archive.Unzip(src, dst)
	if err != nil {
		log.Printf("Unzip error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleCreateSession(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Root       string `json:"root"`
		TTLSeconds int    `json:"ttlSeconds"`
		ReadOnly   bool   `json:"readonly"`
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

	session, err := s.Sessions.CreateSession(absRoot, req.TTLSeconds, req.ReadOnly)
	if err != nil {
		log.Printf("CreateSession error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(session)
}

func (s *Server) SetupRoutes() *http.ServeMux {
	mux := http.NewServeMux()

	// UI API
	mux.HandleFunc("/api/list", s.authMiddleware(s.handleList))
	mux.HandleFunc("/api/file", s.authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			s.handleGetFile(w, r)
		} else if r.Method == http.MethodPut {
			s.handlePutFile(w, r)
		}
	}))
	mux.HandleFunc("/api/upload", s.authMiddleware(s.handleUpload))
	// JSON endpoints with max bytes middleware
	mux.Handle("/api/mkdir", maxBytesMiddleware(MaxRequestSize)(s.authMiddleware(s.handleMkdir)))
	mux.Handle("/api/rename", maxBytesMiddleware(MaxRequestSize)(s.authMiddleware(s.handleRename)))
	mux.Handle("/api/delete", maxBytesMiddleware(MaxRequestSize)(s.authMiddleware(s.handleDelete)))
	mux.Handle("/api/zip", maxBytesMiddleware(MaxRequestSize)(s.authMiddleware(s.handleZip)))
	mux.Handle("/api/extract", maxBytesMiddleware(MaxRequestSize)(s.authMiddleware(s.handleExtract)))

	return mux
}

func (s *Server) SetupManagementRoutes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/sessions", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			s.handleCreateSession(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
	return mux
}
