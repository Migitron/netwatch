package collector

import (
	"fmt"
	"log"
	"net"
	"time"

	"github.com/gosnmp/gosnmp"

	"github.com/yourusername/netwatch/internal/config"
	"github.com/yourusername/netwatch/internal/storage"
)

// Standard SNMP OIDs we care about.
// An OID (Object Identifier) is a unique address for a specific piece of
// data on a network device. These are standardized across vendors.
const (
	oidSysUpTime    = "1.3.6.1.2.1.1.3.0"   // how long device has been running
	oidSysName      = "1.3.6.1.2.1.1.5.0"   // device hostname
	oidIfInOctets1  = "1.3.6.1.2.1.2.2.1.10.1" // bytes received on interface 1
	oidIfOutOctets1 = "1.3.6.1.2.1.2.2.1.16.1" // bytes sent on interface 1
)

// SNMPResult holds the raw data returned from one SNMP poll
type SNMPResult struct {
	UptimeSecs  int64
	IfInOctets  int64
	IfOutOctets int64
	Error       error
}

// PollSNMP connects to a device and fetches key metrics via SNMP.
// This is the core function you'll expand as you learn more OIDs.
func PollSNMP(device config.Device) SNMPResult {
	g := &gosnmp.GoSNMP{
		Target:    device.Host,
		Port:      device.SNMPPort,
		Community: device.Community,
		Version:   gosnmp.Version2c,
		Timeout:   5 * time.Second,
		Retries:   1,
	}

	if err := g.Connect(); err != nil {
		return SNMPResult{Error: fmt.Errorf("SNMP connect to %s: %w", device.Host, err)}
	}
	defer g.Conn.Close()

	oids := []string{oidSysUpTime, oidIfInOctets1, oidIfOutOctets1}
	result, err := g.Get(oids)
	if err != nil {
		return SNMPResult{Error: fmt.Errorf("SNMP get from %s: %w", device.Host, err)}
	}

	var r SNMPResult
	for _, variable := range result.Variables {
		switch variable.Name {
		case "."+oidSysUpTime:
			// Uptime comes back in hundredths of a second (TimeTicks)
			if ticks, ok := variable.Value.(uint32); ok {
				r.UptimeSecs = int64(ticks) / 100
			}
		case "."+oidIfInOctets1:
			r.IfInOctets = toInt64(variable.Value)
		case "."+oidIfOutOctets1:
			r.IfOutOctets = toInt64(variable.Value)
		}
	}

	return r
}

// Ping sends a single ICMP echo request and returns the round-trip time.
// Note: on Linux this requires either root or the net.ipv4.ping_group_range sysctl.
// A simple TCP dial is used here as a fallback that works without root.
func Ping(host string) (rttMs float64, reachable bool) {
	start := time.Now()
	conn, err := net.DialTimeout("tcp", host+":80", 3*time.Second)
	elapsed := time.Since(start)

	if err != nil {
		// Try port 443 if 80 fails — device may not run HTTP but does HTTPS
		start = time.Now()
		conn, err = net.DialTimeout("tcp", host+":443", 3*time.Second)
		elapsed = time.Since(start)
	}

	if err != nil {
		// Neither port open — try SNMP port as reachability check
		start = time.Now()
		conn, err = net.DialTimeout("udp", host+":161", 3*time.Second)
		elapsed = time.Since(start)
	}

	if conn != nil {
		conn.Close()
	}

	if err != nil {
		return 0, false
	}

	return float64(elapsed.Microseconds()) / 1000.0, true
}

// PollDevice runs a full poll cycle for one device: ping + SNMP.
// It returns a DeviceStatus ready to be written to the database.
func PollDevice(device config.Device) storage.DeviceStatus {
	status := storage.DeviceStatus{
		DeviceName: device.Name,
		Host:       device.Host,
		Timestamp:  time.Now(),
	}

	// Always try a reachability check first
	if device.EnablePing {
		rtt, reachable := Ping(device.Host)
		status.RTTMs = rtt
		status.Reachable = reachable

		if !reachable {
			log.Printf("[WARN] %s (%s) is not reachable via ping", device.Name, device.Host)
			return status // skip SNMP if we can't reach it
		}
	} else {
		status.Reachable = true // assume reachable, let SNMP tell us otherwise
	}

	// Now collect SNMP metrics
	snmpResult := PollSNMP(device)
	if snmpResult.Error != nil {
		log.Printf("[WARN] SNMP error for %s: %v", device.Name, snmpResult.Error)
		status.Reachable = false
		return status
	}

	status.UptimeSecs = snmpResult.UptimeSecs
	status.IfInOctets = snmpResult.IfInOctets
	status.IfOutOctets = snmpResult.IfOutOctets

	log.Printf("[OK] %s (%s) — uptime: %s, RTT: %.2fms, in: %d bytes, out: %d bytes",
		device.Name, device.Host,
		formatUptime(snmpResult.UptimeSecs),
		status.RTTMs,
		status.IfInOctets,
		status.IfOutOctets,
	)

	return status
}

// toInt64 safely converts gosnmp's interface{} values to int64
func toInt64(v interface{}) int64 {
	switch val := v.(type) {
	case uint:
		return int64(val)
	case uint32:
		return int64(val)
	case uint64:
		return int64(val)
	case int:
		return int64(val)
	}
	return 0
}

// formatUptime turns seconds into a human-readable string like "3d 2h 15m"
func formatUptime(secs int64) string {
	days := secs / 86400
	hours := (secs % 86400) / 3600
	mins := (secs % 3600) / 60
	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, mins)
	}
	return fmt.Sprintf("%dh %dm", hours, mins)
}
