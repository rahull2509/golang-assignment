# TaskBridge: Cross-Platform Remote Job Runner

> A distributed job execution system built in Go, where a central server orchestrates job lifecycle and one or more agents execute tasks and report results.

## Architecture

```
┌─────────────────────────────────────────────────┐
│                  TaskBridge Server               │
│                                                  │
│  ┌──────────┐  ┌────────────┐  ┌──────────────┐ │
│  │ REST API │  │   Store    │  │   Health     │ │
│  │ Handlers │──│ (Memory /  │  │   Checker    │ │
│  │          │  │  SQLite)   │  │  (goroutine) │ │
│  └──────────┘  └────────────┘  └──────────────┘ │
│       ▲                                          │
└───────│──────────────────────────────────────────┘
        │ HTTP/JSON
        │
┌───────│──────────────────────────────────────────┐
│       ▼            TaskBridge Agent              │
│  ┌──────────┐  ┌────────────┐  ┌──────────────┐ │
│  │  HTTP    │  │   Worker   │  │  Executor    │ │
│  │  Client  │──│   Loop     │──│  Registry    │ │
│  │          │  │ (poll/hb)  │  │  (7 types)   │ │
│  └──────────┘  └────────────┘  └──────────────┘ │
└──────────────────────────────────────────────────┘
```

## Features

- **REST API** with structured JSON request/response DTOs
- **7 job executors**: `http_check`, `tcp_check`, `file_exists`, `checksum`, `copy_file`, `write_file`, `wait`
- **Job lifecycle**: PENDING → RUNNING → SUCCESS/FAILED with automatic retry
- **Agent management**: registration, heartbeat, online/offline detection
- **Capability-based scheduling**: jobs assigned only to compatible agents
- **Concurrency-safe** in-memory store with `sync.RWMutex`
- **Structured logging** via `log/slog`
- **Graceful shutdown** with OS signal handling
- **Auth token** support (optional shared Bearer token)
- **Comprehensive tests** with race detection

## Quick Start

### Prerequisites

- Go 1.22 or later

### Run the Server

```bash
go run ./cmd/server --addr :8080
```

Server flags:
| Flag | Default | Description |
|------|---------|-------------|
| `--addr` | `:8080` | Listen address |
| `--auth-token` | `` | Shared auth token (empty = disabled) |
| `--log-json` | `false` | JSON log format |

### Run the Agent

```bash
go run ./cmd/agent --server http://localhost:8080 --id agent-1
```

Agent flags:
| Flag | Default | Description |
|------|---------|-------------|
| `--server` | `http://localhost:8080` | Server URL |
| `--id` | `agent-dev-1` | Agent identifier |
| `--capabilities` | `http_check,tcp_check,...` | Comma-separated job types |
| `--poll-interval` | `3s` | Job polling interval |
| `--auth-token` | `` | Shared auth token |
| `--log-json` | `false` | JSON log format |

### Build Binaries

```bash
go build -o bin/taskbridge-server ./cmd/server
go build -o bin/taskbridge-agent ./cmd/agent
```

## API Reference

### Health

```bash
curl http://localhost:8080/health
```

### Create a Job

```bash
curl -X POST http://localhost:8080/jobs \
  -H "Content-Type: application/json" \
  -d '{
    "name": "check-app-health",
    "type": "http_check",
    "payload": {
      "url": "http://localhost:8080/health",
      "expected_status": 200
    },
    "timeout_seconds": 10,
    "max_retries": 2
  }'
```

### List All Jobs

```bash
curl http://localhost:8080/jobs
```

### Get a Specific Job

```bash
curl http://localhost:8080/jobs/{jobId}
```

### Cancel a Job

```bash
curl -X POST http://localhost:8080/jobs/{jobId}/cancel
```

### Register an Agent

```bash
curl -X POST http://localhost:8080/agents/register \
  -H "Content-Type: application/json" \
  -d '{
    "id": "agent-1",
    "hostname": "worker-01",
    "os": "linux",
    "arch": "amd64",
    "version": "1.0.0",
    "capabilities": ["http_check", "tcp_check", "wait"]
  }'
```

### Send Heartbeat

```bash
curl -X POST http://localhost:8080/agents/agent-1/heartbeat
```

### Poll for Next Job

```bash
curl -X POST http://localhost:8080/agents/agent-1/next-job \
  -H "Content-Type: application/json" \
  -d '{"capabilities": ["http_check", "wait"]}'
```

### Submit Job Result

```bash
curl -X POST http://localhost:8080/jobs/{jobId}/result \
  -H "Content-Type: application/json" \
  -d '{
    "status": "SUCCESS",
    "logs": ["check passed"],
    "result": {"actual_status": 200}
  }'
```

### List Agents

```bash
curl http://localhost:8080/agents
```

## Job Types

| Type | Payload Keys | Description |
|------|-------------|-------------|
| `http_check` | `url`, `expected_status` | GET request, compare status code |
| `tcp_check` | `address` | Dial TCP host:port |
| `file_exists` | `path` | Check if file exists |
| `checksum` | `path`, `expected_checksum`? | SHA-256 checksum |
| `copy_file` | `source`, `destination` | Copy file |
| `write_file` | `path`, `content`, `mode`? | Write content to file |
| `wait` | `duration_seconds` | Sleep (for testing) |

## Job Lifecycle

```
PENDING ──→ RUNNING ──→ SUCCESS
                    └──→ FAILED ──→ (if retries remain) ──→ PENDING (re-queued)
                                └──→ (retries exhausted) ──→ FAILED (terminal)

Any non-terminal state ──→ CANCELED (via POST /jobs/{id}/cancel)
```

## Project Structure

```
taskbridge/
├── cmd/
│   ├── server/main.go          # Server entry point
│   └── agent/main.go           # Agent entry point
├── internal/
│   ├── model/
│   │   ├── model.go            # Domain entities
│   │   ├── dto.go              # Request/response DTOs
│   │   └── validation.go       # Input validation
│   ├── store/
│   │   ├── store.go            # Store interface
│   │   └── memory.go           # In-memory implementation
│   ├── api/
│   │   ├── server.go           # Router, middleware chain
│   │   ├── helpers.go          # JSON encode/decode
│   │   ├── middleware.go       # Logging, recovery, auth
│   │   ├── health.go           # GET /health
│   │   ├── jobs.go             # Job CRUD handlers
│   │   ├── agents.go           # Agent handlers
│   │   └── assignment.go       # Job assignment + result
│   ├── executor/
│   │   ├── executor.go         # Interface + registry
│   │   ├── http_check.go       # HTTP check
│   │   ├── tcp_check.go        # TCP check
│   │   ├── file_exists.go      # File exists check
│   │   ├── checksum.go         # SHA-256 checksum
│   │   ├── copy_file.go        # File copy
│   │   ├── write_file.go       # File write
│   │   └── wait.go             # Wait/sleep
│   ├── agent/
│   │   ├── client.go           # Agent HTTP client
│   │   └── worker.go           # Poll/heartbeat/execute loop
│   └── config/
│       └── config.go           # Config structs
├── examples/                   # Sample job JSON payloads
├── docs/                       # Documentation
└── Makefile                    # Build/test targets
```

## Testing

```bash
# Run all tests with race detection
go test ./... -v -race

# Run with coverage
go test ./... -v -race -coverprofile=coverage.out
go tool cover -func=coverage.out
```

## Demo

### Full End-to-End Demo

**Terminal 1 — Start Server:**
```bash
go run ./cmd/server --addr :8080
```

**Terminal 2 — Start Agent:**
```bash
go run ./cmd/agent --server http://localhost:8080 --id agent-1
```

**Terminal 3 — Create and Monitor Jobs:**

```bash
# 1. Check server health
curl -s http://localhost:8080/health | jq

# 2. Create an HTTP check job
curl -s -X POST http://localhost:8080/jobs \
  -H "Content-Type: application/json" \
  -d @examples/create-http-check-job.json | jq

# 3. Create a wait job
curl -s -X POST http://localhost:8080/jobs \
  -H "Content-Type: application/json" \
  -d @examples/create-wait-job.json | jq

# 4. List all jobs (watch them go from PENDING → SUCCESS)
curl -s http://localhost:8080/jobs | jq

# 5. List agents
curl -s http://localhost:8080/agents | jq
```

## Demo Video

### YouTube Demo

https://youtu.be/gJXdS8gwR2E

### Google Drive Demo

https://drive.google.com/file/d/14l6bNEzxYdWBjuvB8TR94SwaqSo1tOcI/view?usp=sharing

This demo video showcases:

* Server startup
* Agent registration and heartbeat
* Health endpoint verification
* Job creation and execution
* Multiple supported job types
* Job lifecycle tracking
* Agent polling and scheduling
* End-to-end TaskBridge workflow

## Additional Documentation

* DEMO_GUIDE.md — Step-by-step project execution and demonstration guide.
* TaskBridge_Milestone_Report_With_Screenshots.docx — Milestone evidence document containing console screenshots and explanations.

## Design Decisions

1. **Standard library only** — no external HTTP frameworks (Gin/Echo/Chi) to demonstrate Go fundamentals
2. **Go 1.22 routing** — method-aware patterns (`"GET /jobs/{jobId}"`) for clean routing
3. **Dependency injection** — `Server` receives `Store` interface via constructor for testability
4. **`sync.RWMutex`** — read-heavy workloads benefit from concurrent readers
5. **FIFO job assignment** — preserves creation order for fair scheduling
6. **`context.Context`** — propagated through executors for timeout and cancellation
7. **`log/slog`** — structured logging from the standard library (Go 1.21+)
8. **Retry in store layer** — `CompleteJob` handles FAILED→PENDING transition atomically

## License

This project is part of a Go internship assignment.
