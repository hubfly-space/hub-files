package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"hubfly-files/internal/archive"
	"hubfly-files/internal/config"
	"hubfly-files/internal/filebackend"
	"hubfly-files/internal/filesystem"
	"hubfly-files/internal/hostmount"
	"hubfly-files/internal/sessions"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

const (
	// MaxRequestSize is the maximum allowed request body size (1 MB for JSON)
	MaxRequestSize = 1 << 20
	// Demo storage values exposed in demo mode.
	DemoTotalBytes = 20 << 20
	DemoUsedBytes  = 5 << 20
	DefaultSMBPort = 445
)

type Server struct {
	Config   *config.Config
	Sessions *sessions.Store
	SMBPool  *filebackend.SMBPool
	// Store    *sqlite.Storage
}

func NewServer(cfg *config.Config, store *sessions.Store) *Server {
	return &Server{
		Config:   cfg,
		Sessions: store,
		SMBPool:  filebackend.NewSMBPool(),
		// Store:    db,
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

// func methodHandler(method string, next http.HandlerFunc) http.HandlerFunc {
// 	return func(w http.ResponseWriter, r *http.Request) {
// 		if r.Method != method {
// 			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
// 			return
// 		}
// 		next(w, r)
// 	}
// }

func methodHandlers(methods []string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		if slices.Contains(methods, r.Method) {
			next(w, r)
			return

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

type sessionContextKey struct{}

func sessionFromRequest(r *http.Request) *sessions.Session {
	session, _ := r.Context().Value(sessionContextKey{}).(*sessions.Session)
	return session
}

func (s *Server) backendForRequest(r *http.Request) filebackend.Backend {
	session := sessionFromRequest(r)
	if session != nil && session.SMB != nil {
		return filebackend.SMB{
			Pool:   s.SMBPool,
			Key:    session.Code,
			Config: *session.SMB,
		}
	}
	return filebackend.Local{Root: r.Header.Get("X-Session-Root")}
}

func parseSMBSessionRoot(root, username, password, domain, workstation string) (string, *sessions.SMBConfig, error) {
	u, err := url.Parse(root)
	if err != nil {
		return "", nil, err
	}
	if strings.ToLower(u.Scheme) != "smb" {
		return "", nil, fmt.Errorf("unsupported SMB scheme")
	}

	host := u.Hostname()
	if host == "" {
		return "", nil, fmt.Errorf("SMB host is required")
	}

	port := DefaultSMBPort
	if rawPort := u.Port(); rawPort != "" {
		parsed, err := strconv.Atoi(rawPort)
		if err != nil || parsed <= 0 || parsed > 65535 {
			return "", nil, fmt.Errorf("invalid SMB port")
		}
		port = parsed
	}

	parts := strings.Split(strings.Trim(u.EscapedPath(), "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		return "", nil, fmt.Errorf("SMB share is required")
	}

	share, err := url.PathUnescape(parts[0])
	if err != nil || share == "" || strings.ContainsAny(share, `/\\`) {
		return "", nil, fmt.Errorf("invalid SMB share")
	}

	baseParts := make([]string, 0, len(parts)-1)
	for _, rawPart := range parts[1:] {
		part, err := url.PathUnescape(rawPart)
		if err != nil {
			return "", nil, fmt.Errorf("invalid SMB base path")
		}
		if part == "" || part == "." {
			continue
		}
		if part == ".." || strings.ContainsRune(part, 0) {
			return "", nil, filesystem.ErrUnauthorized
		}
		baseParts = append(baseParts, part)
	}

	if username == "" {
		username = u.User.Username()
	}
	if password == "" {
		password, _ = u.User.Password()
	}
	if domain == "" {
		for _, sep := range []string{";", `\\`} {
			if before, after, ok := strings.Cut(username, sep); ok {
				domain = before
				username = after
				break
			}
		}
	}

	cfg := &sessions.SMBConfig{
		Host:        host,
		Port:        port,
		Share:       share,
		BasePath:    strings.Join(baseParts, "/"),
		Username:    username,
		Password:    password,
		Domain:      domain,
		Workstation: workstation,
	}

	return sanitizedSMBRoot(*cfg), cfg, nil
}

func sanitizedSMBRoot(cfg sessions.SMBConfig) string {
	host := cfg.Host
	if cfg.Port != 0 && cfg.Port != DefaultSMBPort {
		host = net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port))
	}

	parts := []string{cfg.Share}
	if cfg.BasePath != "" {
		parts = append(parts, strings.Split(cfg.BasePath, "/")...)
	}

	return (&url.URL{Scheme: "smb", Host: host, Path: "/" + strings.Join(parts, "/")}).String()
}

func pathpkgBase(filePath string) string {
	return path.Base(strings.ReplaceAll(filePath, `\`, "/"))
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

		ctx := context.WithValue(r.Context(), sessionContextKey{}, session)
		next(w, r.WithContext(ctx))
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
	path := r.URL.Query().Get("path")

	files, err := s.backendForRequest(r).List(r.Context(), path)
	if err != nil {
		log.Printf("List error: %v", err)
		writeFileSystemError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(files)
}

func (s *Server) handleSessionInfo(w http.ResponseWriter, r *http.Request) {
	session := sessionFromRequest(r)
	if session == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	response := struct {
		Root          string `json:"root"`
		Type          string `json:"type"`
		CanHostMount  bool   `json:"canHostMount"`
		HostMountRoot string `json:"hostMountRoot,omitempty"`
		ReadOnly      bool   `json:"readonly"`
		AllowUpload   bool   `json:"allowUpload"`
		AllowEdit     bool   `json:"allowEdit"`
		AllowDelete   bool   `json:"allowDelete"`
	}{
		Root:         session.Root,
		Type:         "local",
		CanHostMount: false,
		ReadOnly:     session.ReadOnly,
		AllowUpload:  session.AllowUpload,
		AllowEdit:    session.AllowEdit,
		AllowDelete:  session.AllowDelete,
	}
	if session.SMB != nil {
		response.Type = "smb"
		response.CanHostMount = s.Config.AllowHostMounts
		if s.Config.AllowHostMounts {
			response.HostMountRoot = s.Config.HostMountRoot
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleHostMount(w http.ResponseWriter, r *http.Request) {
	if !s.Config.AllowHostMounts {
		http.Error(w, "Host mounts are disabled", http.StatusForbidden)
		return
	}

	session := sessionFromRequest(r)
	if session == nil || session.SMB == nil {
		http.Error(w, "Host mount requires an SMB session", http.StatusBadRequest)
		return
	}

	result, err := hostmount.MountSMB(r.Context(), s.Config.HostMountRoot, session.SMB)
	if err != nil {
		log.Printf("Host mount error: %v", err)
		switch {
		case errors.Is(err, hostmount.ErrUnsupportedSession), errors.Is(err, hostmount.ErrUnsafeMountConfig):
			http.Error(w, err.Error(), http.StatusBadRequest)
		case errors.Is(err, os.ErrPermission):
			http.Error(w, "Permission denied", http.StatusForbidden)
		default:
			http.Error(w, "Host mount failed", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
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

	storage, err := s.backendForRequest(r).Storage(r.Context(), path)
	if err != nil {
		log.Printf("Storage error: %v", err)
		writeFileSystemError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(storage)
}

func (s *Server) handleGetFile(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")

	reader, err := s.backendForRequest(r).Read(r.Context(), path)
	if err != nil {
		log.Printf("GetFile error: %v", err)
		writeFileSystemError(w, err)
		return
	}
	defer reader.Close()

	if r.URL.Query().Get("download") == "1" {
		w.Header().Set("Content-Disposition", "attachment; filename=\""+pathpkgBase(path)+"\"")
		w.Header().Set("Content-Type", "application/octet-stream")
	}

	io.Copy(w, reader)
}

func (s *Server) handlePutFile(w http.ResponseWriter, r *http.Request) {
	if !s.checkPermission(w, r, "AllowEdit") {
		return
	}
	path := r.URL.Query().Get("path")

	err := s.backendForRequest(r).Write(r.Context(), path, r.Body)
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

	backend := s.backendForRequest(r)
	maxUploadBytes := s.Config.MaxUploadBytes
	if maxUploadBytes > 0 {
		if r.ContentLength > maxUploadBytes {
			http.Error(w, "File too large", http.StatusRequestEntityTooLarge)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes)
	}

	if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
		s.handleMultipartUpload(w, r, backend)
		return
	}

	dirPath := r.URL.Query().Get("path")
	filename := r.URL.Query().Get("filename")
	if !validUploadFilename(filename) {
		http.Error(w, "Invalid file name", http.StatusBadRequest)
		return
	}

	finalPath := path.Join(dirPath, filename)
	if err := backend.WriteAtomic(r.Context(), finalPath, r.Body); err != nil {
		log.Printf("Upload error: %v", err)
		writeUploadError(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleMultipartUpload(w http.ResponseWriter, r *http.Request, backend filebackend.Backend) {
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

			finalPath := path.Join(dirPath, filename)
			if err := backend.WriteAtomic(r.Context(), finalPath, part); err != nil {
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
	var req struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	err := s.backendForRequest(r).Mkdir(r.Context(), req.Path)
	if err != nil {
		log.Printf("Mkdir error: %v", err)
		writeFileSystemError(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleTouch(w http.ResponseWriter, r *http.Request) {
	if s.checkReadOnly(w, r) {
		return
	}
	var req struct {
		Path string `json:"path"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request Body", http.StatusBadRequest)
		return
	}

	err := s.backendForRequest(r).Touch(r.Context(), req.Path)
	if err != nil {
		log.Printf("Touch error: %v", err)
		writeFileSystemError(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)

}
func (s *Server) handleRename(w http.ResponseWriter, r *http.Request) {
	if s.checkReadOnly(w, r) {
		return
	}
	var req struct {
		OldPath string `json:"oldPath"`
		NewPath string `json:"newPath"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	err := s.backendForRequest(r).Rename(r.Context(), req.OldPath, req.NewPath)
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
	path := r.URL.Query().Get("path")

	err := s.backendForRequest(r).Delete(r.Context(), path)
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
	if session := sessionFromRequest(r); session != nil && session.SMB != nil {
		http.Error(w, "zip is not supported for SMB sessions yet", http.StatusNotImplemented)
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
	if session := sessionFromRequest(r); session != nil && session.SMB != nil {
		http.Error(w, "extract is not supported for SMB sessions yet", http.StatusNotImplemented)
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
		Root           string `json:"root"`
		TTLSeconds     int    `json:"ttlSeconds"`
		ReadOnly       bool   `json:"readonly"`
		AllowUpload    bool   `json:"allowUpload"`
		AllowEdit      bool   `json:"allowEdit"`
		AllowDelete    bool   `json:"allowDelete"`
		SMBUsername    string `json:"smbUsername"`
		SMBPassword    string `json:"smbPassword"`
		SMBDomain      string `json:"smbDomain"`
		SMBWorkstation string `json:"smbWorkstation"`
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

	root := req.Root
	var smbCfg *sessions.SMBConfig
	var err error
	if strings.EqualFold(strings.TrimSpace(root), "smb://") || strings.HasPrefix(strings.ToLower(root), "smb://") {
		root, smbCfg, err = parseSMBSessionRoot(root, req.SMBUsername, req.SMBPassword, req.SMBDomain, req.SMBWorkstation)
		if err != nil {
			log.Printf("CreateSession SMB root error: %v", err)
			http.Error(w, "Invalid SMB root", http.StatusBadRequest)
			return
		}
	} else {
		// Clean and validate the local root path.
		cleanRoot := filepath.Clean(root)
		root, err = filepath.Abs(cleanRoot)
		if err != nil {
			http.Error(w, "Invalid root path", http.StatusBadRequest)
			return
		}
	}

	var session *sessions.Session
	if smbCfg != nil {
		session, err = s.Sessions.CreateSMBSession(root, req.TTLSeconds, req.ReadOnly, req.AllowUpload, req.AllowEdit, req.AllowDelete, smbCfg)
	} else {
		session, err = s.Sessions.CreateSession(root, req.TTLSeconds, req.ReadOnly, req.AllowUpload, req.AllowEdit, req.AllowDelete)
	}
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
	mux.HandleFunc("/api/session", s.authMiddleware(methodHandlers([]string{http.MethodGet}, s.handleSessionInfo)))
	mux.HandleFunc("/api/host-mount", s.authMiddleware(methodHandlers([]string{http.MethodPost}, s.handleHostMount)))
	mux.HandleFunc("/api/list", s.authMiddleware(methodHandlers([]string{http.MethodGet}, s.handleList)))
	mux.HandleFunc("/api/storage", s.authMiddleware(methodHandlers([]string{http.MethodGet}, s.handleStorage)))
	mux.HandleFunc("/api/file", s.authMiddleware(methodHandlers([]string{http.MethodGet, http.MethodPut}, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			s.handleGetFile(w, r)
			return
		}
		s.handlePutFile(w, r)
	})))
	mux.HandleFunc("/api/upload", s.authMiddleware(methodHandlers([]string{http.MethodPost}, s.handleUpload)))
	// JSON endpoints with max bytes middleware
	mux.Handle("/api/mkdir", maxBytesMiddleware(MaxRequestSize)(s.authMiddleware(methodHandlers([]string{http.MethodPost}, s.handleMkdir))))
	mux.Handle("/api/rename", maxBytesMiddleware(MaxRequestSize)(s.authMiddleware(methodHandlers([]string{http.MethodPost}, s.handleRename))))
	mux.Handle("/api/delete", maxBytesMiddleware(MaxRequestSize)(s.authMiddleware(methodHandlers([]string{http.MethodDelete}, s.handleDelete))))
	mux.Handle("/api/zip", maxBytesMiddleware(MaxRequestSize)(s.authMiddleware(methodHandlers([]string{http.MethodPost}, s.handleZip))))
	mux.Handle("/api/extract", maxBytesMiddleware(MaxRequestSize)(s.authMiddleware(methodHandlers([]string{http.MethodPost}, s.handleExtract))))
	mux.Handle("/api/touch", maxBytesMiddleware(MaxRequestSize)(s.authMiddleware(methodHandlers([]string{http.MethodPost}, s.handleTouch))))
	mux.HandleFunc("/", s.handleUI)

	return mux
}

func (s *Server) SetupManagementRoutes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/sessions", methodHandlers([]string{http.MethodPost}, s.handleCreateSession))
	return mux
}
