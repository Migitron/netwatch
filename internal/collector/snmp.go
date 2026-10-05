package collector

import (
	"time"

	"github.com/gosnmp/gosnmp"
	"github.com/migiton/netwatch/internal/config"
)

// OIDs from the SNMP system group (RFC 1213). The leading dot matches the
// form gosnmp uses in responses, so the same constants work for both.
const (
	oidSysDescr  = ".1.3.6.1.2.1.1.1.0"
	oidSysUpTime = ".1.3.6.1.2.1.1.3.0"
	oidSysName   = ".1.3.6.1.2.1.1.5.0"
)

const (
	snmpTimeout = 2 * time.Second
	snmpRetries = 1
)

// SystemInfo holds the SNMP system values read from a single device.
type SystemInfo struct {
	Device    string // name from the config file
	IP        string
	SysName   string // name the device reports about itself
	Descr     string
	Uptime    time.Duration
	Err       error
	CheckedAt time.Time
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

	res, err := g.Get([]string{oidSysDescr, oidSysUpTime, oidSysName})
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
		}
	}
	return info
}
