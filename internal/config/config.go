// Package config loads and parses the netwatch YAML config file.
package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

const (
	defaultSNMPPort  = 161
	defaultCommunity = "public"
)

// Config is the top-level structure of the config file.
type Config struct {
	Port    int      `yaml:"port"`
	Devices []Device `yaml:"devices"`
}

// Device is a single host to monitor.
type Device struct {
	Name       string `yaml:"name"`
	Host       string `yaml:"host"`      // IP address or hostname
	Community  string `yaml:"community"` // SNMP v2c community string
	SNMPPort   uint16 `yaml:"snmp_port"`
	EnablePing bool   `yaml:"enable_ping"`
}

// Load reads the config file at path and fills in defaults for any
// device fields that were left empty.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config file %q: %w", path, err)
	}

	cfg.applyDefaults()
	return &cfg, nil
}

func (c *Config) applyDefaults() {
	for i := range c.Devices {
		d := &c.Devices[i]
		if d.SNMPPort == 0 {
			d.SNMPPort = defaultSNMPPort
		}
		if d.Community == "" {
			d.Community = defaultCommunity
		}
	}
}
