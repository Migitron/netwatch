package storage

import (
	"database/sql"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// DB wraps out SQLite connection
type DB struct {
	conn *sql.DB
}

// DeviceStatus is what is stored every poll cycle
type DeviceStatus struct {
	DeviceName  string
	Host        string
	Timestamp   time.Time
	Reachable   bool
	RTTMs       float64 // ping round-trip time in milliseconds
	UptimeSecs  int64   // from SNMP sysUpTime
	IfInOctets  int64   // interface bytes in (index 1)
	IfOutOctets int64   // interface bytes out (index 1)
}
