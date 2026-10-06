// Command netwatch pings and polls (via SNMP) the devices listed in its
// config file and prints the results.
package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/migiton/netwatch/internal/collector"
	"github.com/migiton/netwatch/internal/config"
)

func main() {
	configPath := flag.String("config", "netwatch.yaml", "path to config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	fmt.Println("Netwatching port:", cfg.Port)

	printPingResults(collector.PingAll(cfg))
	printPollResults(collector.PollAll(cfg))
}

func printPingResults(results []collector.PingResult) {
	fmt.Println("Pinging all devices:")
	for _, r := range results {
		switch {
		case r.Skipped:
			fmt.Printf("%s | IP = %s: SKIPPED (ping disabled)\n", r.Device, r.IP)
		case r.Err != nil:
			fmt.Printf("%s | IP = %s: ERROR: %v\n", r.Device, r.IP, r.Err)
		case r.Up:
			fmt.Printf("%s | IP = %s: Status = UP rtt = %v loss = %.0f%%\n", r.Device, r.IP, r.AvgRTT, r.PacketLoss)
		default:
			fmt.Printf("%s | IP = %s: Status = DOWN (no replies)\n", r.Device, r.IP)
		}
	}
}

func printPollResults(results []collector.SystemInfo) {
	fmt.Println("Polling all devices:")
	for _, r := range results {
		switch {
		case r.Skipped:
			fmt.Printf("%s | IP = %s: SKIPPED (SNMP disabled)\n", r.Device, r.IP)
		case r.Err != nil:
			fmt.Printf("%s | IP = %s: SNMP ERROR: %v\n", r.Device, r.IP, r.Err)
		default:
			fmt.Printf("%s | IP = %s: sysName = %q UP:%v IN:%v OUT:%v\n", r.Device, r.IP, r.SysName, r.Uptime.Round(time.Second), r.IfInOctets, r.IfOutOctets)
		}
	}
}
