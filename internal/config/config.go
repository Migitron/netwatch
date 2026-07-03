// This packaage loads the config file and returns the parsed file
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Device struct {
	Name       string `yaml:"name"`
	Host       string `yaml:"host"`      // IP address or hostname
	Community  string `yaml:"community"` // SNMP community string (usually "public")
	SNMPPort   uint16 `yaml:"snmp_port"` // usually 161
	EnablePing bool   `yaml:"enable_ping"`
}

type Config struct {
	Port    int
	Devices []Device
}

func Load(path string) (*Config, error) {
	if !filepath.IsAbs(path) {
		var err error
		path, err = filepath.Abs(path)
		if err != nil {
			return nil, fmt.Errorf("error resolving config path %q: %w", path, err)
		}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	var cfg Config
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		return nil, fmt.Errorf("error parsing config YAML: %w", err)
	}

	return &cfg, nil
}
