package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/migiton/netwatch/internal/collector"
	"github.com/migiton/netwatch/internal/config"
)

func main() {
	configPath := flag.String("config", "netwatch.yaml", "path to config file") //allows users to set a config path with flag --config
	flag.Parse()                                                                //reads the flags from os.Args

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("[FATAL] Failed to load config: %v", err)
	}

	fmt.Println("Netwatching port: ", cfg.Port)

	collector.Ping(cfg)
	collector.PollSystem(cfg)
}
