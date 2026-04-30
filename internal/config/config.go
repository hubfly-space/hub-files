package config

import (
	"flag"
	"os"
)

type Config struct {
	APIPort        string
	ManagementPort string
	DemoDir        string
}

func LoadConfig() *Config {
	apiPort := flag.String("api-port", "8080", "Port for UI/API server")
	mgmtPort := flag.String("mgmt-port", "9090", "Port for Management server")
	demoDir := flag.String("demo-dir", "./demo", "Directory for demo mode")

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

	return &Config{
		APIPort:        *apiPort,
		ManagementPort: *mgmtPort,
		DemoDir:        *demoDir,
	}
}
