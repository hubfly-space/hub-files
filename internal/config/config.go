package config

import (
	"flag"
	"os"
	"path/filepath"
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
	APIPort             string
	ManagementPort      string
	HealthPort          string
	DemoDir             string
	UIDir               string
	MaxUploadBytes      int64
	AllowHostMounts     bool
	HostMountRoot       string
	HostMountConfigRoot string
	HostMountUID        int
	HostMountGID        int
	HostMountUmask      string
}

func LoadConfig() *Config {
	apiPort := flag.String("api-port", "10015", "Port for UI/API server")
	mgmtPort := flag.String("mgmt-port", "10014", "Port for Management server")
	healthPort := flag.String("health-port", "60003", "Port for health check server")
	demoDir := flag.String("demo-dir", "./demo", "Directory for demo mode")
	uiDir := flag.String("ui-dir", "./frontend/dist", "Directory containing built UI assets")
	maxUploadBytes := flag.Int64("max-upload-bytes", 100<<20, "Maximum upload size in bytes (0 disables the limit)")
	allowHostMounts := flag.Bool("allow-host-mounts", false, "Allow remote sessions to mount on this host")
	hostMountRoot := flag.String("host-mount-root", "/mnt/hubfiles", "Directory for host mounts")
	hostMountConfigRoot := flag.String("host-mount-config-root", defaultHostMountConfigRoot(), "Private directory for host mount config and credentials")
	hostMountUID := flag.Int("host-mount-uid", defaultHostMountUID(), "UID that should own host-mounted files")
	hostMountGID := flag.Int("host-mount-gid", defaultHostMountGID(), "GID that should own host-mounted files")
	hostMountUmask := flag.String("host-mount-umask", "002", "Umask for host-mounted files")

	flag.Parse()

	// Allow environment variables to override flags
	if p := firstEnv("HUBFILES_API_PORT", "HUBFLY_API_PORT"); p != "" {
		*apiPort = p
	}
	if p := firstEnv("HUBFILES_MGMT_PORT", "HUBFLY_MGMT_PORT"); p != "" {
		*mgmtPort = p
	}
	if p := firstEnv("HUBFILES_HEALTH_PORT", "HUBFLY_HEALTH_PORT"); p != "" {
		*healthPort = p
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
	if d := firstEnv("HUBFILES_HOST_MOUNT_CONFIG_ROOT", "HUBFLY_HOST_MOUNT_CONFIG_ROOT"); d != "" {
		*hostMountConfigRoot = d
	}
	if v := firstEnv("HUBFILES_HOST_MOUNT_UID", "HUBFLY_HOST_MOUNT_UID"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed >= 0 {
			*hostMountUID = parsed
		}
	}
	if v := firstEnv("HUBFILES_HOST_MOUNT_GID", "HUBFLY_HOST_MOUNT_GID"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed >= 0 {
			*hostMountGID = parsed
		}
	}
	if v := firstEnv("HUBFILES_HOST_MOUNT_UMASK", "HUBFLY_HOST_MOUNT_UMASK"); v != "" {
		*hostMountUmask = v
	}

	return &Config{
		APIPort:             *apiPort,
		ManagementPort:      *mgmtPort,
		HealthPort:          *healthPort,
		DemoDir:             *demoDir,
		UIDir:               *uiDir,
		MaxUploadBytes:      *maxUploadBytes,
		AllowHostMounts:     *allowHostMounts,
		HostMountRoot:       *hostMountRoot,
		HostMountConfigRoot: *hostMountConfigRoot,
		HostMountUID:        *hostMountUID,
		HostMountGID:        *hostMountGID,
		HostMountUmask:      *hostMountUmask,
	}
}

func defaultHostMountConfigRoot() string {
	if configDir, err := os.UserConfigDir(); err == nil && configDir != "" {
		return filepath.Join(configDir, "hubfiles", "hostmount")
	}
	if homeDir, err := os.UserHomeDir(); err == nil && homeDir != "" {
		return filepath.Join(homeDir, ".config", "hubfiles", "hostmount")
	}
	return filepath.Join(os.TempDir(), "hubfiles-hostmount")
}

func defaultHostMountUID() int {
	if value := os.Getenv("SUDO_UID"); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil && parsed >= 0 {
			return parsed
		}
	}
	return os.Getuid()
}

func defaultHostMountGID() int {
	if value := os.Getenv("SUDO_GID"); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil && parsed >= 0 {
			return parsed
		}
	}
	return os.Getgid()
}
