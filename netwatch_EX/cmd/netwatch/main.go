package main

import (
	"flag"
	"log"
	"sync"
	"time"

	"github.com/yourusername/netwatch/internal/alerts"
	"github.com/yourusername/netwatch/internal/api"
	"github.com/yourusername/netwatch/internal/collector"
	"github.com/yourusername/netwatch/internal/config"
	"github.com/yourusername/netwatch/internal/storage"
)

func main() {
	// Parse command-line flags
	configPath := flag.String("config", "netwatch.yaml", "path to config file")
	listenAddr := flag.String("listen", ":8080", "address for the web dashboard")
	flag.Parse()

	log.Println("[INFO] NetWatch starting up...")

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("[FATAL] Failed to load config: %v", err)
	}
	log.Printf("[INFO] Loaded %d device(s) from config", len(cfg.Devices))

	// Open the database
	db, err := storage.Open(cfg.DBPath)
	if err != nil {
		log.Fatalf("[FATAL] Failed to open database: %v", err)
	}
	defer db.Close()
	log.Printf("[INFO] Database ready at %s", cfg.DBPath)

	// Set up the alert engine
	alertEngine := alerts.NewEngine(
		cfg.Alerts.SlackWebhookURL,
		cfg.Alerts.OfflineAfterSec,
		db,
	)

	// Start the HTTP server in a background goroutine so it doesn't block polling
	server := api.NewServer(db, "dashboard/static")
	go func() {
		if err := server.Start(*listenAddr); err != nil {
			log.Fatalf("[FATAL] HTTP server error: %v", err)
		}
	}()

	// Run the polling loop forever
	pollInterval := time.Duration(cfg.PollInterval) * time.Second
	log.Printf("[INFO] Polling every %s", pollInterval)

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	// Poll once immediately on startup, then on each tick
	runPollCycle(cfg, db, alertEngine)

	for range ticker.C {
		runPollCycle(cfg, db, alertEngine)
	}
}

// runPollCycle polls all configured devices concurrently.
// This is a key Go concept: instead of polling devices one-by-one (slow),
// we launch one goroutine per device so they all run in parallel.
// The WaitGroup lets us wait until every goroutine has finished.
func runPollCycle(cfg *config.Config, db *storage.DB, engine *alerts.Engine) {
	log.Printf("[INFO] Starting poll cycle for %d device(s)...", len(cfg.Devices))

	var wg sync.WaitGroup

	for _, device := range cfg.Devices {
		wg.Add(1)
		// Capture device in loop variable (important in Go — look up "goroutine loop variable capture")
		dev := device
		go func() {
			defer wg.Done()

			status := collector.PollDevice(dev)

			if err := db.WriteStatus(status); err != nil {
				log.Printf("[ERROR] Failed to write status for %s: %v", dev.Name, err)
			}

			engine.Evaluate(status)
		}()
	}

	wg.Wait()
	log.Println("[INFO] Poll cycle complete")
}
