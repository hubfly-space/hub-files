package server

import (
	"encoding/json"
	"hubfly-files/internal/archive"
	"hubfly-files/internal/config"
	"hubfly-files/internal/filesystem"
	"hubfly-files/internal/sessions"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
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

// Middleware to validate session
func (s *Server) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if strings.HasPrefix(token, "Bearer ") {
			token = strings.TrimPrefix(token, "Bearer ")
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

		// Inject session into context or just pass it to handlers
		// For simplicity, we'll pass it in the request header or just use it here
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleUpload(w http.ResponseWriter, r *http.Request) {
	if s.checkReadOnly(w, r) {
		return
	}
	root := r.Header.Get("X-Session-Root")
	path := r.FormValue("path")

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	finalPath := filepath.Join(path, header.Filename)
	err = filesystem.WriteFile(root, finalPath, file)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
	json.NewDecoder(r.Body).Decode(&req)

	err := filesystem.Mkdir(root, req.Path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
	json.NewDecoder(r.Body).Decode(&req)

	err := filesystem.Rename(root, req.OldPath, req.NewPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
	json.NewDecoder(r.Body).Decode(&req)

	src, _ := filesystem.SafePath(root, req.Source)
	dst, _ := filesystem.SafePath(root, req.Target)

	err := archive.Zip(src, dst)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
	json.NewDecoder(r.Body).Decode(&req)

	src, _ := filesystem.SafePath(root, req.Source)
	dst, _ := filesystem.SafePath(root, req.Target)

	err := archive.Unzip(src, dst)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	session, err := s.Sessions.CreateSession(req.Root, req.TTLSeconds, req.ReadOnly)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
	mux.HandleFunc("/api/mkdir", s.authMiddleware(s.handleMkdir))
	mux.HandleFunc("/api/rename", s.authMiddleware(s.handleRename))
	mux.HandleFunc("/api/delete", s.authMiddleware(s.handleDelete))
	mux.HandleFunc("/api/zip", s.authMiddleware(s.handleZip))
	mux.HandleFunc("/api/extract", s.authMiddleware(s.handleExtract))

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
