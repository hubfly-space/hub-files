package config

import (
	"flag"
	"os"
)

type Config struct {
	APIPort        string
	ManagementPort string
	DemoDir        string
	UIDir          string
}

func LoadConfig() *Config {
	apiPort := flag.String("api-port", "10015", "Port for UI/API server")
	mgmtPort := flag.String("mgmt-port", "10014", "Port for Management server")
	demoDir := flag.String("demo-dir", "./demo", "Directory for demo mode")
	uiDir := flag.String("ui-dir", "./frontend/dist", "Directory containing built UI assets")

	flag.Parse()

	// Allow environment variables to override flags
	if p := os.Getenv("HUBFLY_API_PORT"); p != "" {
		*apiPort = p
	}
	if p := os.Getenv("HUBFLY_MGMT_PORT"); p != "" {
		*mgmtPort = p
	}
	if d := os.Getenv("HUBFLY_DEMO_DIR"); d != "" {
		*demoDir = d
	}
	if d := os.Getenv("HUBFLY_UI_DIR"); d != "" {
		*uiDir = d
	}

	return &Config{
		APIPort:        *apiPort,
		ManagementPort: *mgmtPort,
		DemoDir:        *demoDir,
		UIDir:          *uiDir,
	}
}
