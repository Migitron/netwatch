# NetWatch

A network infrastructure monitoring tool built in Go. Polls network devices via SNMP and ICMP, stores metrics in SQLite, and displays a live web dashboard.

Built as a portfolio project to demonstrate Go concurrency, networking protocols, REST API design, and database usage — grounded in real infrastructure engineering experience.

---

## Architecture

```
netwatch.yaml (config)
       │
       ▼
┌─────────────────┐     goroutine per device
│   main.go       │ ──────────────────────────►  collector.PollDevice()
│   (poll loop)   │                                    │ SNMP + Ping
└────────┬────────┘                                    │
         │                                             ▼
         │                                      storage.WriteStatus()
         │                                             │ SQLite
         ▼                                             │
  alerts.Evaluate()  ◄──────────────────────────────────
         │ state change?
         ▼
  Slack webhook / log

  api.Server (HTTP)
         │
         ├── GET /api/devices          → latest status per device
         ├── GET /api/devices/{n}/history → historical metrics
         ├── GET /api/alerts           → recent alert events
         └── GET /                     → dashboard HTML
```

## Project Structure

```
netwatch/
├── cmd/netwatch/main.go        # Entry point — wires everything together
├── internal/
│   ├── config/config.go        # YAML config loading
│   ├── collector/collector.go  # SNMP polling + ping
│   ├── storage/storage.go      # SQLite read/write
│   ├── alerts/alerts.go        # Alert state machine
│   └── api/api.go              # HTTP REST API
├── dashboard/static/
│   └── index.html              # Web dashboard
├── netwatch.yaml               # Your config (edit this)
└── README.md
```

## Prerequisites

- Go 1.22+
- GCC (required by the SQLite driver): `sudo apt install gcc` on Ubuntu, or use the pure-Go driver (see below)
- A network device with SNMP enabled, OR any Linux machine with `snmpd` installed

## Quick Start

```bash
# 1. Clone and enter the project
git clone https://github.com/yourusername/netwatch
cd netwatch

# 2. Download dependencies
go mod tidy

# 3. Edit the config for your environment
nano netwatch.yaml

# 4. Build and run
go run ./cmd/netwatch -config netwatch.yaml

# 5. Open the dashboard
open http://localhost:8080
```

## Setting Up a Test Device (No Hardware Required)

You can test entirely with software using `snmpd` on Linux or WSL:

```bash
# Install the SNMP daemon on Ubuntu/Debian
sudo apt install snmpd

# Edit the config to listen on localhost
sudo nano /etc/snmp/snmpd.conf
# Change: agentAddress  udp:127.0.0.1:161
# Add:    rocommunity public localhost

sudo systemctl restart snmpd

# Now set host: "127.0.0.1" in netwatch.yaml and run NetWatch
```

## Key Go Concepts Demonstrated

| Concept | Where |
|---|---|
| Goroutines + WaitGroup | `main.go` — concurrent device polling |
| Mutexes | `alerts/alerts.go` — protecting shared state |
| Interfaces | `storage.DB` could implement an interface for testing |
| Error wrapping (`%w`) | Throughout — Go 1.13+ error handling |
| HTTP server (stdlib) | `api/api.go` — no framework needed |
| Defer for cleanup | `collector.go`, `storage.go` |
| Struct tags | `config/config.go` — YAML unmarshaling |

## What I'd Add Next (Phase 2)

- [ ] WebSocket feed for real-time push updates (no polling from browser)
- [ ] Prometheus `/metrics` endpoint for Grafana integration
- [ ] Config hot-reload using `fsnotify` (no restart needed)
- [ ] Unit tests for the alert state machine
- [ ] Docker + docker-compose for one-command deployment
- [ ] Topology map showing device relationships

## Why I Built This

I work in infrastructure engineering — designing physical network deployments. I wanted to deeply understand the software side of what I physically install: how monitoring tools actually work, how SNMP translates to data, how concurrent polling works at scale. This project bridges both worlds.
