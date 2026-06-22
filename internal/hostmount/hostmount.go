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
	ErrUnsupportedSession = errors.New("host mount requires an SMB session")
	ErrUnsafeMountConfig  = errors.New("unsafe SMB mount config")
)

type Result struct {
	MountPath      string `json:"mountPath"`
	AlreadyMounted bool   `json:"alreadyMounted"`
}

func MountSMB(ctx context.Context, mountRoot string, cfg *sessions.SMBConfig) (*Result, error) {
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

	credentialsPath, err := writeCredentials(mountRoot, *cfg)
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

func writeCredentials(mountRoot string, cfg sessions.SMBConfig) (string, error) {
	credentialsDir := filepath.Join(mountRoot, ".credentials")
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
