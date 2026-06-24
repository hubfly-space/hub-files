package hostmount

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"hubfly-files/internal/sessions"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

var (
	ErrUnsupportedSession = errors.New("host mount requires an SMB or FTP session")
	ErrUnsafeMountConfig  = errors.New("unsafe host mount config")
)

type Result struct {
	MountPath      string `json:"mountPath"`
	AlreadyMounted bool   `json:"alreadyMounted"`
}

type UnmountResult struct {
	MountPath  string `json:"mountPath"`
	WasMounted bool   `json:"wasMounted"`
}

type Options struct {
	UID   int
	GID   int
	Umask string
}

func MountSMB(ctx context.Context, mountRoot, configRoot string, cfg *sessions.SMBConfig, opts Options) (*Result, error) {
	if cfg == nil {
		return nil, ErrUnsupportedSession
	}
	if err := validateConfig(cfg); err != nil {
		return nil, err
	}

	mountPath := MountPath(mountRoot, *cfg)
	if err := os.MkdirAll(mountPath, 0755); err != nil {
		return nil, err
	}

	mounted, err := IsMounted(mountPath)
	if err != nil {
		return nil, err
	}
	if mounted {
		return &Result{MountPath: mountPath, AlreadyMounted: true}, nil
	}

	credentialsPath, err := writeCredentials(configRoot, *cfg)
	if err != nil {
		return nil, err
	}

	options := []string{
		"credentials=" + credentialsPath,
		"iocharset=utf8",
		"vers=3.0",
		"noserverino",
	}
	if cfg.Port > 0 && cfg.Port != 445 {
		options = append(options, "port="+strconv.Itoa(cfg.Port))
	}
	if cfg.BasePath != "" {
		options = append(options, "prefixpath="+cfg.BasePath)
	}

	source := "//" + cfg.Host + "/" + cfg.Share
	cmd := exec.CommandContext(ctx, "mount", "-t", "cifs", source, mountPath, "-o", strings.Join(options, ","))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("mount.cifs failed: %w: %s", err, strings.TrimSpace(string(output)))
	}

	return &Result{MountPath: mountPath}, nil
}

func MountFTP(ctx context.Context, mountRoot, configRoot string, cfg *sessions.FTPConfig, opts Options) (*Result, error) {
	if cfg == nil {
		return nil, ErrUnsupportedSession
	}
	if err := validateFTPConfig(cfg); err != nil {
		return nil, err
	}

	mountPath := FTPMountPath(mountRoot, *cfg)
	if err := os.MkdirAll(mountPath, 0755); err != nil {
		return nil, err
	}

	mounted, err := IsMounted(mountPath)
	if err != nil {
		return nil, err
	}
	if mounted {
		return &Result{MountPath: mountPath, AlreadyMounted: true}, nil
	}

	configPath, remoteName, err := writeRcloneFTPConfig(ctx, configRoot, *cfg)
	if err != nil {
		return nil, err
	}

	remote := remoteName + ":"
	if cfg.BasePath != "" {
		remote += "/" + strings.Trim(strings.TrimPrefix(cfg.BasePath, "/"), "/")
	}

	args := []string{
		"mount",
		remote,
		mountPath,
		"--config", configPath,
		"--vfs-cache-mode", "writes",
		"--dir-cache-time", "30s",
		"--poll-interval", "0",
		"--daemon",
	}
	if opts.UID >= 0 {
		args = append(args, "--uid", strconv.Itoa(opts.UID))
	}
	if opts.GID >= 0 {
		args = append(args, "--gid", strconv.Itoa(opts.GID))
	}
	if opts.Umask != "" {
		args = append(args, "--umask", opts.Umask)
	}
	args = append(args, "--allow-other")

	cmd := exec.CommandContext(ctx, "rclone", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("rclone mount failed: %w: %s", err, strings.TrimSpace(string(output)))
	}

	return &Result{MountPath: mountPath}, nil
}

func UnmountSMB(ctx context.Context, mountRoot string, cfg *sessions.SMBConfig) (*UnmountResult, error) {
	if cfg == nil {
		return nil, ErrUnsupportedSession
	}
	if err := validateConfig(cfg); err != nil {
		return nil, err
	}
	return unmountPath(ctx, MountPath(mountRoot, *cfg))
}

func UnmountFTP(ctx context.Context, mountRoot string, cfg *sessions.FTPConfig) (*UnmountResult, error) {
	if cfg == nil {
		return nil, ErrUnsupportedSession
	}
	if err := validateFTPConfig(cfg); err != nil {
		return nil, err
	}
	return unmountPath(ctx, FTPMountPath(mountRoot, *cfg))
}

func unmountPath(ctx context.Context, mountPath string) (*UnmountResult, error) {
	mounted, err := IsMounted(mountPath)
	if err != nil {
		return nil, err
	}
	if !mounted {
		if err := removeMountPoint(mountPath); err != nil {
			return nil, err
		}
		return &UnmountResult{MountPath: mountPath, WasMounted: false}, nil
	}

	cmd := exec.CommandContext(ctx, "fusermount3", "-u", mountPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		cmd = exec.CommandContext(ctx, "fusermount", "-u", mountPath)
		output, err = cmd.CombinedOutput()
	}
	if err != nil {
		cmd = exec.CommandContext(ctx, "umount", mountPath)
		output, err = cmd.CombinedOutput()
	}
	if err != nil {
		return nil, fmt.Errorf("unmount failed: %w: %s", err, strings.TrimSpace(string(output)))
	}
	if err := removeMountPoint(mountPath); err != nil {
		return nil, err
	}

	return &UnmountResult{MountPath: mountPath, WasMounted: true}, nil
}

func removeMountPoint(mountPath string) error {
	if err := os.Remove(mountPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove mount point failed: %w", err)
	}
	return nil
}

func FTPMountPath(mountRoot string, cfg sessions.FTPConfig) string {
	sum := sha256.Sum256([]byte(strings.Join([]string{
		cfg.Host,
		strconv.Itoa(cfg.Port),
		cfg.BasePath,
		cfg.Username,
	}, "\x00")))
	hash := hex.EncodeToString(sum[:])[:12]
	name := sanitizeName(cfg.Host + "-ftp")
	return filepath.Join(mountRoot, name+"-"+hash)
}

func MountPath(mountRoot string, cfg sessions.SMBConfig) string {
	sum := sha256.Sum256([]byte(strings.Join([]string{
		cfg.Host,
		strconv.Itoa(cfg.Port),
		cfg.Share,
		cfg.BasePath,
		cfg.Domain,
		cfg.Username,
	}, "\x00")))
	hash := hex.EncodeToString(sum[:])[:12]
	name := sanitizeName(cfg.Host + "-" + cfg.Share)
	return filepath.Join(mountRoot, name+"-"+hash)
}

func IsMounted(mountPath string) (bool, error) {
	file, err := os.Open("/proc/self/mountinfo")
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	defer file.Close()

	cleanMountPath, err := filepath.Abs(mountPath)
	if err != nil {
		return false, err
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 5 && unescapeMountInfo(fields[4]) == cleanMountPath {
			return true, nil
		}
	}
	return false, scanner.Err()
}

func validateConfig(cfg *sessions.SMBConfig) error {
	values := []string{cfg.Host, cfg.Share, cfg.BasePath, cfg.Username, cfg.Password, cfg.Domain, cfg.Workstation}
	for _, value := range values {
		if strings.ContainsAny(value, "\x00\n\r") {
			return ErrUnsafeMountConfig
		}
	}
	if cfg.Host == "" || cfg.Share == "" {
		return ErrUnsafeMountConfig
	}
	if strings.ContainsAny(cfg.Host, "/\\,") || strings.ContainsAny(cfg.Share, "/\\,") {
		return ErrUnsafeMountConfig
	}
	if strings.Contains(cfg.BasePath, ",") {
		return ErrUnsafeMountConfig
	}
	return nil
}

func writeCredentials(configRoot string, cfg sessions.SMBConfig) (string, error) {
	credentialsDir := filepath.Join(configRoot, "credentials")
	if err := os.MkdirAll(credentialsDir, 0700); err != nil {
		return "", err
	}

	credentialsPath := filepath.Join(credentialsDir, sanitizeName(cfg.Host+"-"+cfg.Share)+".cred")
	content := "username=" + cfg.Username + "\n" + "password=" + cfg.Password + "\n"
	if cfg.Domain != "" {
		content += "domain=" + cfg.Domain + "\n"
	}
	if err := os.WriteFile(credentialsPath, []byte(content), 0600); err != nil {
		return "", err
	}
	return credentialsPath, nil
}

func sanitizeName(input string) string {
	input = strings.ToLower(input)
	var b strings.Builder
	for _, r := range input {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		} else {
			b.WriteByte('-')
		}
	}
	name := strings.Trim(b.String(), "-")
	if name == "" {
		return "smb-share"
	}
	return name
}

func unescapeMountInfo(input string) string {
	replacer := strings.NewReplacer(`\040`, " ", `\011`, "\t", `\012`, "\n", `\134`, `\`)
	return replacer.Replace(input)
}

func validateFTPConfig(cfg *sessions.FTPConfig) error {
	values := []string{cfg.Host, cfg.BasePath, cfg.Username, cfg.Password}
	for _, value := range values {
		if strings.ContainsAny(value, "\x00\n\r") {
			return ErrUnsafeMountConfig
		}
	}
	if cfg.Host == "" || cfg.Username == "" {
		return ErrUnsafeMountConfig
	}
	if strings.ContainsAny(cfg.Host, "/\\,") || strings.Contains(cfg.BasePath, ",") {
		return ErrUnsafeMountConfig
	}
	return nil
}

func writeRcloneFTPConfig(ctx context.Context, configRoot string, cfg sessions.FTPConfig) (string, string, error) {
	configDir := filepath.Join(configRoot, "rclone")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return "", "", err
	}

	remoteName := sanitizeName(cfg.Host+"-ftp") + "-" + rcloneRemoteHash(cfg)
	configPath := filepath.Join(configDir, remoteName+".conf")
	obscuredPassword, err := obscureRclonePassword(ctx, cfg.Password)
	if err != nil {
		return "", "", err
	}
	content := "[" + remoteName + "]\n" +
		"type = ftp\n" +
		"host = " + cfg.Host + "\n" +
		"user = " + cfg.Username + "\n" +
		"pass = " + obscuredPassword + "\n"
	if cfg.Port > 0 && cfg.Port != 21 {
		content += "port = " + strconv.Itoa(cfg.Port) + "\n"
	}

	if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
		return "", "", err
	}
	return configPath, remoteName, nil
}

func obscureRclonePassword(ctx context.Context, password string) (string, error) {
	cmd := exec.CommandContext(ctx, "rclone", "obscure", password)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("rclone obscure failed: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return strings.TrimSpace(string(output)), nil
}

func rcloneRemoteHash(cfg sessions.FTPConfig) string {
	sum := sha256.Sum256([]byte(strings.Join([]string{
		cfg.Host,
		strconv.Itoa(cfg.Port),
		cfg.BasePath,
		cfg.Username,
	}, "\x00")))
	return hex.EncodeToString(sum[:])[:12]
}
