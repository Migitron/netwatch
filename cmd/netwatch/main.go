package main

import (
	"fmt"
	"log"

	"github.com/migitron/netwatch/internal/config"
)

func main() {
	path := "netwatch.yaml"

	cfg, err := config.Load(path)
	if err != nil {
		log.Fatalf("[FATAL] Failed to load config: %v", err)
	}

	fmt.Println("Netwatching port: ", cfg.path)
	for _, dev := range cfg.Devices {
		fmt.Printf("Name: %s | IP: %s \n", dev.Name, dev.Host)
	}

}
