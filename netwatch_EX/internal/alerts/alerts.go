package alerts

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/yourusername/netwatch/internal/storage"
)

// Engine tracks device states and fires alerts when they change.
// This is a good example of why Go's concurrency model matters:
// the engine is accessed from multiple goroutines (one per device poller)
// so we protect shared state with a mutex.
type Engine struct {
	webhookURL      string
	offlineAfterSec int

	mu           sync.Mutex              // protects the maps below
	lastSeen     map[string]time.Time    // last time device was reachable
	deviceStates map[string]bool         // true = online, false = offline
	db           *storage.DB
}

// NewEngine creates a new alert engine
func NewEngine(webhookURL string, offlineAfterSec int, db *storage.DB) *Engine {
	return &Engine{
		webhookURL:      webhookURL,
		offlineAfterSec: offlineAfterSec,
		lastSeen:        make(map[string]time.Time),
		deviceStates:    make(map[string]bool),
		db:              db,
	}
}

// Evaluate is called after every poll. It checks if the device's state has
// changed (online → offline or offline → online) and fires alerts accordingly.
func (e *Engine) Evaluate(status storage.DeviceStatus) {
	e.mu.Lock()
	defer e.mu.Unlock()

	prevOnline, known := e.deviceStates[status.DeviceName]

	if status.Reachable {
		e.lastSeen[status.DeviceName] = time.Now()

		// Device just came back online
		if known && !prevOnline {
			e.deviceStates[status.DeviceName] = true
			msg := fmt.Sprintf("✅ *%s* (%s) is back ONLINE", status.DeviceName, status.Host)
			log.Println("[ALERT]", msg)
			e.db.WriteAlert(status.DeviceName, status.Host, "online", msg)
			e.sendSlack(msg)
		} else {
			e.deviceStates[status.DeviceName] = true
		}
		return
	}

	// Device is not reachable — check if it's been down long enough to alert
	last, hasSeen := e.lastSeen[status.DeviceName]
	downDuration := time.Since(last)

	if !hasSeen {
		// Never seen before — mark as offline but don't alert yet
		e.deviceStates[status.DeviceName] = false
		return
	}

	if downDuration >= time.Duration(e.offlineAfterSec)*time.Second {
		// Device just crossed the threshold — only alert on the transition
		if prevOnline || !known {
			e.deviceStates[status.DeviceName] = false
			msg := fmt.Sprintf("🔴 *%s* (%s) has been OFFLINE for %s",
				status.DeviceName, status.Host, downDuration.Round(time.Second))
			log.Println("[ALERT]", msg)
			e.db.WriteAlert(status.DeviceName, status.Host, "offline", msg)
			e.sendSlack(msg)
		}
	}
}

// sendSlack posts a message to a Slack incoming webhook.
// If no webhook is configured, this is a no-op.
func (e *Engine) sendSlack(message string) {
	if e.webhookURL == "" {
		return
	}

	payload := map[string]string{"text": message}
	body, _ := json.Marshal(payload)

	resp, err := http.Post(e.webhookURL, "application/json", bytes.NewReader(body))
	if err != nil {
		log.Printf("[WARN] Slack webhook failed: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("[WARN] Slack webhook returned status %d", resp.StatusCode)
	}
}
