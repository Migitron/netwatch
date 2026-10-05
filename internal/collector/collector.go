// Package collector gathers reachability (ICMP ping) and system information
// (SNMP) from the devices listed in the config.
package collector

import (
	"sync"

	"github.com/migiton/netwatch/internal/config"
)

// PingAll pings every configured device concurrently. Results are returned in
// the same order as cfg.Devices.
func PingAll(cfg *config.Config) []PingResult {
	results := make([]PingResult, len(cfg.Devices))
	var wg sync.WaitGroup
	for i, dev := range cfg.Devices {
		if !dev.EnablePing {
			results[i] = PingResult{Device: dev.Name, IP: dev.Host, Skipped: true}
			continue
		}
		wg.Go(func() {
			results[i] = pingHost(dev)
		})
	}
	wg.Wait()
	return results
}

// PollAll queries every configured device over SNMP concurrently. Results are
// returned in the same order as cfg.Devices.
func PollAll(cfg *config.Config) []SystemInfo {
	results := make([]SystemInfo, len(cfg.Devices))
	var wg sync.WaitGroup
	for i, dev := range cfg.Devices {
		if !dev.EnableSNMP {
			results[i] = SystemInfo{Device: dev.Name, IP: dev.Host, Skipped: true}
			continue
		}
		wg.Go(func() {
			results[i] = pollSystem(dev)
		})
	}
	wg.Wait()
	return results
}
