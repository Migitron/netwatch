package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/migiton/netwatch/internal/config"

	probing "github.com/prometheus-community/pro-bing"
)

type PingResult struct {
	IP         string
	Up         bool
	AvgRTT     time.Duration
	PacketLoss float64
	Err        error
	CheckedAt  time.Time
}

func pingHost(ip string, count int, timeout time.Duration) PingResult {
	res := PingResult{IP: ip, CheckedAt: time.Now()}

	pinger, err := probing.NewPinger(ip)
	if err != nil {
		res.Err = err // bad address of DNS failer
		return res
	}
	pinger.Count = count
	pinger.Timeout = timeout    //caps total tine for the whole run
	pinger.SetPrivileged(false) // works unprivileged on macOS

	if err := pinger.Run(); err != nil {
		res.Err = err
		return res
	}
	stats := pinger.Statistics()
	res.Up = stats.PacketsRecv > 0
	res.AvgRTT = stats.AvgRtt
	res.PacketLoss = stats.PacketLoss
	return res

}

func main() {
	configPath := flag.String("config", "netwatch.yaml", "path to config file") //allows users to set a config path with flag --config
	flag.Parse()                                                                //reads the flags from os.Args

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("[FATAL] Failed to load config: %v", err)
	}

	fmt.Println("Netwatching port: ", cfg.Port)
	for _, dev := range cfg.Devices {
		fmt.Printf("Name: %s | IP: %s \n", dev.Name, dev.Host)
	}

	for _, ips := range cfg.Devices {
		result := pingHost(ips.Host, 3, 3*time.Second)
		if result.Up {
			fmt.Println("ping successful")
		} else {
			fmt.Println("ping UNsuccessful")
		}
	}

}
