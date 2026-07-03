package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config is the top-level structure loaded from netwatch.yaml
type Config struct {
	PollInterval int      `yaml:"poll_interval_seconds"` // how often to poll devices
	DBPath       string   `yaml:"db_path"`               // path to SQLite file
	Devices      []Device `yaml:"devices"`
	Alerts       Alerts   `yaml:"alerts"`
}

// Device represents one monitored network device
type Device struct {
	Name      string `yaml:"name"`
	Host      string `yaml:"host"`       // IP address or hostname
	Community string `yaml:"community"`  // SNMP community string (usually "public")
	SNMPPort  uint16 `yaml:"snmp_port"`  // usually 161
	EnablePing bool  `yaml:"enable_ping"`
}

// Alerts holds alert destination config
type Alerts struct {
	SlackWebhookURL string `yaml:"slack_webhook_url"`
	OfflineAfterSec int    `yaml:"offline_after_seconds"` // how long before firing alert
}

// Load reads and parses the YAML config file at the given path
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config YAML: %w", err)
	}

	// Set sensible defaults if not specified
	if cfg.PollInterval == 0 {
		cfg.PollInterval = 30
	}
	if cfg.DBPath == "" {
		cfg.DBPath = "netwatch.db"
	}
	if cfg.Alerts.OfflineAfterSec == 0 {
		cfg.Alerts.OfflineAfterSec = 120
	}

	return &cfg, nil
}
