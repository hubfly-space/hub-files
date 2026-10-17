package config

import (
	"flag"
	"os"
	"testing"
)

func TestHealthBindAddress(t *testing.T) {
	oldFlags, oldArgs := flag.CommandLine, os.Args
	defer func() { flag.CommandLine = oldFlags; os.Args = oldArgs }()
	flag.CommandLine = flag.NewFlagSet("test", flag.ContinueOnError)
	os.Args = []string{"test"}
	t.Setenv("HUBFILES_HEALTH_ADDR", "100.66.1.2:60003")
	cfg := LoadConfig()
	if cfg.HealthAddr != "100.66.1.2:60003" || cfg.HealthPort != "60003" {
		t.Fatal(cfg.HealthAddr, cfg.HealthPort)
	}
}
