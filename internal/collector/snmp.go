package collector

import (
	"time"

	"github.com/gosnmp/gosnmp"
	"github.com/migiton/netwatch/internal/config"
)

// OIDs from the SNMP system group (RFC 1213). The leading dot matches the
// form gosnmp uses in responses, so the same constants work for both.
const (
	oidSysDescr     = ".1.3.6.1.2.1.1.1.0"
	oidSysUpTime    = ".1.3.6.1.2.1.1.3.0"     // how long the device has been running
	oidSysName      = ".1.3.6.1.2.1.1.5.0"     //device host name
	oidIfInOctets1  = "1.3.6.1.2.1.2.2.1.10.1" // bytes received on interface 1
	oidIfOutOctets1 = "1.3.6.1.2.1.2.2.1.16.1" // bytes sent on interface 1
)

const (
	snmpTimeout = 2 * time.Second
	snmpRetries = 1
)

// SystemInfo holds the SNMP system values read from a single device.
type SystemInfo struct {
	Device      string // name from the config file
	IP          string
	Skipped     bool   // SNMP is disabled for this device in the config
	SysName     string // name the device reports about itself
	Descr       string
	Uptime      time.Duration
	IfInOctets  int64
	IfOutOctets int64
	Err         error
	CheckedAt   time.Time
}

func pollSystem(dev config.Device) SystemInfo {
	info := SystemInfo{Device: dev.Name, IP: dev.Host, CheckedAt: time.Now()}

	g := &gosnmp.GoSNMP{
		Target:    dev.Host,
		Port:      dev.SNMPPort,
		Community: dev.Community,
		Version:   gosnmp.Version2c,
		Timeout:   snmpTimeout,
		Retries:   snmpRetries,
	}
	if err := g.Connect(); err != nil {
		info.Err = err
		return info
	}
	defer g.Conn.Close()

	oids := []string{oidSysDescr, oidSysUpTime, oidSysName, oidIfInOctets1, oidIfOutOctets1}
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
			// sysUpTime is reported in hundredths of a second.
			if ticks, ok := v.Value.(uint32); ok {
				info.Uptime = time.Duration(ticks) * 10 * time.Millisecond
			}
		case oidSysName:
			if b, ok := v.Value.([]byte); ok {
				info.SysName = string(b)
			}
		case "." + oidIfInOctets1:
			info.IfInOctets = toInt64(v.Value)
		case "." + oidIfOutOctets1:
			info.IfOutOctets = toInt64(v.Value)
		}
	}
	return info
}

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
