package collector

import (
	"fmt"
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

func Ping(cfg *config.Config) error {
	for _, dev := range cfg.Devices {
		fmt.Printf("Name: %s | IP: %s \n", dev.Name, dev.Host)
		result := pingHost(dev.Host, 3, 3*time.Second)
		if result.Up {
			fmt.Println("Ping Successful")
		} else {
			fmt.Println("un-successful")
		}
		// TODO pull out println from the collector and put back into the main.go
		// TODO figure out time it took to ping and return to be printed in main.go
		// TODO create go routine to perform all ping so it is not waiting on one that is not working as fast.
	}
	return nil
}
