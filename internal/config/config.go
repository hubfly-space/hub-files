package config

import (
	"flag"
	"os"
	"strconv"
)

func firstEnv(names ...string) string {
	for _, name := range names {
		if value := os.Getenv(name); value != "" {
			return value
		}
	}
	return ""
}

type Config struct {
	APIPort         string
	ManagementPort  string
	DemoDir         string
	UIDir           string
	MaxUploadBytes  int64
	AllowHostMounts bool
	HostMountRoot   string
}

func LoadConfig() *Config {
	apiPort := flag.String("api-port", "10015", "Port for UI/API server")
	mgmtPort := flag.String("mgmt-port", "10014", "Port for Management server")
	demoDir := flag.String("demo-dir", "./demo", "Directory for demo mode")
	uiDir := flag.String("ui-dir", "./frontend/dist", "Directory containing built UI assets")
	maxUploadBytes := flag.Int64("max-upload-bytes", 100<<20, "Maximum upload size in bytes (0 disables the limit)")
	allowHostMounts := flag.Bool("allow-host-mounts", true, "Allow SMB sessions to mount shares on this host")
	hostMountRoot := flag.String("host-mount-root", "/mnt/hubfiles", "Directory for SMB host mounts")

	flag.Parse()

	// Allow environment variables to override flags
	if p := firstEnv("HUBFILES_API_PORT", "HUBFLY_API_PORT"); p != "" {
		*apiPort = p
	}
	if p := firstEnv("HUBFILES_MGMT_PORT", "HUBFLY_MGMT_PORT"); p != "" {
		*mgmtPort = p
	}
	if d := firstEnv("HUBFILES_DEMO_DIR", "HUBFLY_DEMO_DIR"); d != "" {
		*demoDir = d
	}
	if d := firstEnv("HUBFILES_UI_DIR", "HUBFLY_UI_DIR"); d != "" {
		*uiDir = d
	}
	if n := firstEnv("HUBFILES_MAX_UPLOAD_BYTES", "HUBFLY_MAX_UPLOAD_BYTES"); n != "" {
		if parsed, err := strconv.ParseInt(n, 10, 64); err == nil && parsed >= 0 {
			*maxUploadBytes = parsed
		}
	}
	if v := firstEnv("HUBFILES_ALLOW_HOST_MOUNTS", "HUBFLY_ALLOW_HOST_MOUNTS"); v != "" {
		if parsed, err := strconv.ParseBool(v); err == nil {
			*allowHostMounts = parsed
		}
	}
	if d := firstEnv("HUBFILES_MOUNT_ROOT", "HUBFLY_MOUNT_ROOT"); d != "" {
		*hostMountRoot = d
	}

	return &Config{
		APIPort:         *apiPort,
		ManagementPort:  *mgmtPort,
		DemoDir:         *demoDir,
		UIDir:           *uiDir,
		MaxUploadBytes:  *maxUploadBytes,
		AllowHostMounts: *allowHostMounts,
		HostMountRoot:   *hostMountRoot,
	}
}
