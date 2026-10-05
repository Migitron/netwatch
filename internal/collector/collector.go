package collector

import (
	"fmt"
	"sync"
	"time"

	"github.com/gosnmp/gosnmp"
	"github.com/migiton/netwatch/internal/config"
	probing "github.com/prometheus-community/pro-bing"
)

const (
	oidSysDescr  = ".1.3.6.1.2.1.1.1.0"
	oidSysUpTime = ".1.3.6.1.2.1.1.3.0"
	oidSysName   = ".1.3.6.1.2.1.1.5.0"
)

type PingResult struct {
	Device     string
	IP         string
	Up         bool
	AvgRTT     time.Duration
	PacketLoss float64
	Err        error
	CheckedAt  time.Time
}

type SystemInfo struct {
	IP        string
	Name      string
	Descr     string
	Uptime    time.Duration
	Err       error
	CheckedAt time.Time
}

func pingHost(dev config.Device, count int, timeout time.Duration) PingResult {
	res := PingResult{Device: dev.Name, IP: dev.Host, CheckedAt: time.Now()}

	pinger, err := probing.NewPinger(dev.Host)
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

func PingAll(cfg *config.Config) []PingResult {
	results := make([]PingResult, len(cfg.Devices))
	var wg sync.WaitGroup
	for i, dev := range cfg.Devices {
		wg.Add(1)
		if dev.EnablePing != true {
			continue
		}
		go func(i int, ip string) {
			defer wg.Done()
			results[i] = pingHost(dev, 3, 3*time.Second)
		}(i, dev.Host)
	}
	wg.Wait()
	return results
}

func Ping(cfg *config.Config) {
	fmt.Println("Pinging All Devices:")
	for _, dev := range PingAll(cfg) {
		switch {
		case dev.Err != nil:
			fmt.Printf("%s | IP = %s: ERROR: %v\n", dev.Device, dev.IP, dev.Err)
		case dev.Up:
			fmt.Printf("%s | IP = %s: Status = UP rtt = %v loss = %.0f%%\n", dev.Device, dev.IP, dev.AvgRTT, dev.PacketLoss)
		default:
			fmt.Printf("%s | IP = %s: Status = DOWN (no replies)\n", dev.Device, dev.IP)
		}
	}
}

func PollAllSystems(cfg *config.Config) []SystemInfo {
	results := make([]SystemInfo, len(cfg.Devices))
	var wg sync.WaitGroup

	for i, dev := range cfg.Devices {
		wg.Add(1)
		go func(i int, ip string) {
			defer wg.Done()
			results[i] = PollSystem(dev)
		}(i, dev.Host)
	}

	wg.Wait()
	return results
}

func Poll(cfg *config.Config) {
	fmt.Println("Polling all devices:")
	for _, info := range PollAllSystems(cfg) {
		if info.Err != nil {
			fmt.Printf("%s: SNMP error: %v\n", info.Name, info.IP, info.Err)
			continue
		}
		fmt.Printf("%s : %s up %v\n", info.Name, info.IP, info.Uptime.Round(time.Second))
	}
}

func PollSystem(dev config.Device) SystemInfo {
	info := SystemInfo{Name: dev.Name, IP: dev.Host, CheckedAt: time.Now()}

	g := &gosnmp.GoSNMP{
		Target:    dev.Host,
		Port:      dev.SNMPPort,
		Community: dev.Community,
		Version:   gosnmp.Version2c,
		Timeout:   2 * time.Second,
		Retries:   1,
	}

	if err := g.Connect(); err != nil {
		info.Err = err
		return info
	}
	defer g.Conn.Close()

	oids := []string{
		oidSysDescr, oidSysUpTime, oidSysName, // sysName
	}
	res, err := g.Get(oids)
	if err != nil {
		info.Err = err
		return info
	}

	for _, v := range res.Variables {
		switch v.Name {
		case oidSysDescr:
			if b, ok := v.Value.([]byte); ok {
				info.Descr = string(b)
			}
		case oidSysUpTime:
			if t, ok := v.Value.(uint32); ok {
				info.Uptime = time.Duration(t) * 10 * time.Millisecond
			}
		case oidSysName:
			if b, ok := v.Value.([]byte); ok {
				info.Name = string(b)
			}
		}
	}
	return info
}
