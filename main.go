package main

import (
	"fmt"
	"log"
	"os"

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

func main() {

	data, err := os.ReadFile("netwatch.yaml")
	if err != nil {
		log.Fatalf("reading config file: %w", err)
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		log.Fatalf("parsing config YAML: %w", err)
	}

	fmt.Println("Netwatching port: ", config.Port)
	for _, dev := range config.Devices {
		fmt.Printf("Name: %s | IP: %s \n", dev.Name, dev.Host)
	}

}
