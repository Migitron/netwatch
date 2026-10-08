package storage

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

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

// DB wraps out SQLite connection
type DB struct {
	conn *sql.DB
}

func Open(path string) (*DB, error) {
	conn, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}

	err = conn.Ping()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("connection to %s: %w", path, err)
	}

	db := &DB{conn: conn}

	err = db.migrate()
	if err != nil {
		conn.Close()
		return nil, err
	}
	return db, nil

}

// Migrate creates the tabels if they do not exist
func (db *DB) migrate() error {
	devicesTable := `CREATE TABLE IF NOT EXISTS devices (
	id           INTEGER PRIMARY KEY,
	device_name  TEXT NOT NULL,
	host         TEXT NOT NULL UNIQUE
	)STRICT;`

	metricsTable := `CREATE TABLE IF NOT EXISTS metrics (
    id            INTEGER PRIMARY KEY,
    device_id     INTEGER NOT NULL REFERENCES devices(id),
    timestamp     INTEGER NOT NULL,
    reachable     INTEGER NOT NULL CHECK (reachable IN (0, 1)),
    rtt_ms        REAL,
    uptime_secs   INTEGER,
    if_in_octets  INTEGER,
    if_out_octets INTEGER
	) STRICT;`

	indexTable := `CREATE INDEX IF NOT EXISTS idx_metrics_device_time
    ON metrics (device_id, timestamp);`

	_, err := db.conn.Exec("PRAGMA foreign_keys = ON;")
	if err != nil {
		return err
	}

	_, err = db.conn.Exec(devicesTable)
	if err != nil {
		return err
	}

	_, err = db.conn.Exec(metricsTable)
	if err != nil {
		return err
	}

	_, err = db.conn.Exec(indexTable)
	if err != nil {
		return err
	}
	return nil
}

func (db *DB) Close() error {
	return db.conn.Close()
}
