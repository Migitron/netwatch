package collector

import (
	"net"
	"time"
)

func Ping(host string) (float64, bool) {
	start := time.Now()
	conn, err := net.DialTimeout("tcp", host+":22", 3*time.Second)
	elapsed := time.Since(start)

	if conn != nil {
		conn.Close()
	}

	if err != nil {
		return 0, false
	}
	return float64(elapsed.Microseconds()) / 1000.0, true
}
