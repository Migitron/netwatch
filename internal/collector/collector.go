package collector

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gosnmp/gosnmp"
	"github.com/migiton/netwatch/internal/config"
	probing "github.com/prometheus-community/pro-bing"
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

func pingHost(name string, ip string, count int, timeout time.Duration) PingResult {
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
	res.Device = name
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
		go func(i int, ip string) {
			defer wg.Done()
			results[i] = pingHost(dev.Name, dev.Host, 3, 3*time.Second)
		}(i, dev.Host)
	}
	wg.Wait()
	return results
}

func Ping(cfg *config.Config) {
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

// todo feed in the list of devices from yaml to check everything on the list.
// TODO make this similar to Ping all then call it from another function add return of info
func PollSystem(cfg *config.Config) {

	g := &gosnmp.GoSNMP{
		Target:    dev.Host,
		Port:      dev.SNMPPort,
		Community: dev.Community,
		Version:   gosnmp.Version2c,
		Timeout:   2 * time.Second,
		Retries:   1,
	}

	if err := g.Connect(); err != nil {
		log.Fatal(err)
	}
	defer g.Conn.Close()

	oids := []string{
		"1.3.6.1.2.1.1.1.0", // sysDescr
		"1.3.6.1.2.1.1.3.0", // sysUpTime
		"1.3.6.1.2.1.1.5.0", // sysName
	}
	res, err := g.Get(oids)
	if err != nil {
		log.Fatal(err)
	}

	for _, v := range res.Variables {
		switch v.Type {
		case gosnmp.OctetString:
			fmt.Printf("%s = %s\n", v.Name, string(v.Value.([]byte)))
		case gosnmp.TimeTicks:
			ticks := v.Value.(uint32)
			fmt.Printf("%s = %v\n", v.Name, time.Duration(ticks)*10*time.Millisecond)
		case gosnmp.NoSuchObject, gosnmp.NoSuchInstance:
			fmt.Printf("%s not available on this device\n", v.Name)
		default:
			fmt.Printf("%s = %v\n", v.Name, v.Value)
		}
	}
}
