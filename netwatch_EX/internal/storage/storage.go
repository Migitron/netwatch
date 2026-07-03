package storage

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// DB wraps our SQLite connection
type DB struct {
	conn *sql.DB
}

// DeviceStatus is what we store every poll cycle
type DeviceStatus struct {
	DeviceName    string
	Host          string
	Timestamp     time.Time
	Reachable     bool
	RTTMs         float64 // ping round-trip time in milliseconds
	UptimeSecs    int64   // from SNMP sysUpTime
	IfInOctets    int64   // interface bytes in (index 1)
	IfOutOctets   int64   // interface bytes out (index 1)
}

// Open creates or opens the SQLite database and runs migrations
func Open(path string) (*DB, error) {
	conn, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	db := &DB{conn: conn}
	if err := db.migrate(); err != nil {
		return nil, fmt.Errorf("running migrations: %w", err)
	}
	return db, nil
}

// migrate creates tables if they don't exist.
// In a real project you'd use a migration library, but this is clear and simple.
func (db *DB) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS device_status (
		id            INTEGER PRIMARY KEY AUTOINCREMENT,
		device_name   TEXT NOT NULL,
		host          TEXT NOT NULL,
		timestamp     DATETIME NOT NULL,
		reachable     BOOLEAN NOT NULL,
		rtt_ms        REAL,
		uptime_secs   INTEGER,
		if_in_octets  INTEGER,
		if_out_octets INTEGER
	);

	-- Index so dashboard queries by device+time are fast
	CREATE INDEX IF NOT EXISTS idx_device_time
		ON device_status(device_name, timestamp DESC);

	CREATE TABLE IF NOT EXISTS alert_events (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
		device_name TEXT NOT NULL,
		host        TEXT NOT NULL,
		event_type  TEXT NOT NULL,   -- "offline" or "online"
		timestamp   DATETIME NOT NULL,
		message     TEXT
	);
	`
	_, err := db.conn.Exec(schema)
	return err
}

// WriteStatus inserts a new poll result into device_status
func (db *DB) WriteStatus(s DeviceStatus) error {
	_, err := db.conn.Exec(`
		INSERT INTO device_status
			(device_name, host, timestamp, reachable, rtt_ms, uptime_secs, if_in_octets, if_out_octets)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		s.DeviceName, s.Host, s.Timestamp.UTC(), s.Reachable,
		s.RTTMs, s.UptimeSecs, s.IfInOctets, s.IfOutOctets,
	)
	return err
}

// WriteAlert records an alert event (device went offline or came back)
func (db *DB) WriteAlert(deviceName, host, eventType, message string) error {
	_, err := db.conn.Exec(`
		INSERT INTO alert_events (device_name, host, event_type, timestamp, message)
		VALUES (?, ?, ?, ?, ?)`,
		deviceName, host, eventType, time.Now().UTC(), message,
	)
	return err
}

// LatestStatuses returns the most recent status for every device
func (db *DB) LatestStatuses() ([]DeviceStatus, error) {
	rows, err := db.conn.Query(`
		SELECT device_name, host, timestamp, reachable, rtt_ms, uptime_secs, if_in_octets, if_out_octets
		FROM device_status
		WHERE id IN (
			SELECT MAX(id) FROM device_status GROUP BY device_name
		)
		ORDER BY device_name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []DeviceStatus
	for rows.Next() {
		var s DeviceStatus
		if err := rows.Scan(
			&s.DeviceName, &s.Host, &s.Timestamp, &s.Reachable,
			&s.RTTMs, &s.UptimeSecs, &s.IfInOctets, &s.IfOutOctets,
		); err != nil {
			return nil, err
		}
		results = append(results, s)
	}
	return results, rows.Err()
}

// HistoryForDevice returns the last N status records for a specific device
func (db *DB) HistoryForDevice(deviceName string, limit int) ([]DeviceStatus, error) {
	rows, err := db.conn.Query(`
		SELECT device_name, host, timestamp, reachable, rtt_ms, uptime_secs, if_in_octets, if_out_octets
		FROM device_status
		WHERE device_name = ?
		ORDER BY timestamp DESC
		LIMIT ?
	`, deviceName, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []DeviceStatus
	for rows.Next() {
		var s DeviceStatus
		if err := rows.Scan(
			&s.DeviceName, &s.Host, &s.Timestamp, &s.Reachable,
			&s.RTTMs, &s.UptimeSecs, &s.IfInOctets, &s.IfOutOctets,
		); err != nil {
			return nil, err
		}
		results = append(results, s)
	}
	return results, rows.Err()
}

// RecentAlerts returns the last N alert events
func (db *DB) RecentAlerts(limit int) ([]map[string]interface{}, error) {
	rows, err := db.conn.Query(`
		SELECT device_name, host, event_type, timestamp, message
		FROM alert_events
		ORDER BY timestamp DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var deviceName, host, eventType, message string
		var timestamp time.Time
		if err := rows.Scan(&deviceName, &host, &eventType, &timestamp, &message); err != nil {
			return nil, err
		}
		results = append(results, map[string]interface{}{
			"device_name": deviceName,
			"host":        host,
			"event_type":  eventType,
			"timestamp":   timestamp,
			"message":     message,
		})
	}
	return results, rows.Err()
}

// Close shuts down the database connection
func (db *DB) Close() error {
	return db.conn.Close()
}
