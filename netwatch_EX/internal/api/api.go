package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/yourusername/netwatch/internal/storage"
)

// Server holds the dependencies our HTTP handlers need
type Server struct {
	db   *storage.DB
	mux  *http.ServeMux
}

// NewServer wires up all routes and returns a ready-to-run server
func NewServer(db *storage.DB, dashboardPath string) *Server {
	s := &Server{db: db, mux: http.NewServeMux()}

	// API routes
	s.mux.HandleFunc("GET /api/devices", s.handleDevices)
	s.mux.HandleFunc("GET /api/devices/{name}/history", s.handleDeviceHistory)
	s.mux.HandleFunc("GET /api/alerts", s.handleAlerts)

	// Serve the dashboard's static files (HTML, CSS, JS)
	s.mux.Handle("/", http.FileServer(http.Dir(dashboardPath)))

	return s
}

// Start begins listening on the given address (e.g. ":8080")
func (s *Server) Start(addr string) error {
	log.Printf("[INFO] NetWatch dashboard running at http://localhost%s", addr)
	return http.ListenAndServe(addr, s.mux)
}

// handleDevices returns the latest status for all monitored devices
func (s *Server) handleDevices(w http.ResponseWriter, r *http.Request) {
	statuses, err := s.db.LatestStatuses()
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		log.Printf("[ERROR] LatestStatuses: %v", err)
		return
	}
	writeJSON(w, statuses)
}

// handleDeviceHistory returns historical metrics for one device
func (s *Server) handleDeviceHistory(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")

	limitStr := r.URL.Query().Get("limit")
	limit := 50 // default
	if limitStr != "" {
		if n, err := strconv.Atoi(limitStr); err == nil && n > 0 {
			limit = n
		}
	}

	history, err := s.db.HistoryForDevice(name, limit)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		log.Printf("[ERROR] HistoryForDevice(%s): %v", name, err)
		return
	}
	writeJSON(w, history)
}

// handleAlerts returns recent alert events
func (s *Server) handleAlerts(w http.ResponseWriter, r *http.Request) {
	alerts, err := s.db.RecentAlerts(50)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		log.Printf("[ERROR] RecentAlerts: %v", err)
		return
	}
	writeJSON(w, alerts)
}

// writeJSON is a small helper that sets the right content type and encodes the response
func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*") // handy for local dev
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("[ERROR] JSON encode: %v", err)
	}
}
