package collector

import (
	"time"

	"github.com/migiton/netwatch/internal/config"
	probing "github.com/prometheus-community/pro-bing"
)

const (
	pingCount   = 3
	pingTimeout = 3 * time.Second // caps the total time for the whole run
)

// PingResult is the outcome of pinging a single device.
type PingResult struct {
	Device     string // name from the config file
	IP         string
	Skipped    bool // ping is disabled for this device in the config
	Up         bool // at least one reply was received
	AvgRTT     time.Duration
	PacketLoss float64 // percentage, 0–100
	Err        error
	CheckedAt  time.Time
}

func pingHost(dev config.Device) PingResult {
	res := PingResult{Device: dev.Name, IP: dev.Host, CheckedAt: time.Now()}

	pinger, err := probing.NewPinger(dev.Host)
	if err != nil {
		res.Err = err // invalid address or DNS failure
		return res
	}
	pinger.Count = pingCount
	pinger.Timeout = pingTimeout
	pinger.SetPrivileged(false) // unprivileged ICMP works on macOS without root

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
