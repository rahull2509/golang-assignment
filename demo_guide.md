# TaskBridge Demo Guide

This guide provides a complete walkthrough for running and demonstrating the TaskBridge assignment.

---

# Prerequisites

* Go 1.22+
* Windows PowerShell (or Git Bash)
* Repository cloned locally

Verify Go installation:

```powershell
go version
```

---

# Terminal Layout

Open **3 terminals**.

### Terminal 1 — Server

Used to run the TaskBridge server.

### Terminal 2 — Agent

Used to run the TaskBridge worker agent.

### Terminal 3 — API Testing

Used to create jobs, verify health, and inspect results.

---

# Step 1: Start Server

Open Terminal 1:

```powershell
go run ./cmd/server --addr :8080
```

Expected output:

```text
TaskBridge server starting
addr=:8080
auth=false
```

Server should remain running.

---

# Step 2: Start Agent

Open Terminal 2:

```powershell
go run ./cmd/agent --server http://localhost:8080 --id agent-1
```

Expected output:

```text
agent registered
heartbeat sent
polling for jobs
```

The agent should continuously poll the server and send heartbeats.

---

# Step 3: Verify Health Endpoint

Open Terminal 3:

```powershell
Invoke-RestMethod http://localhost:8080/health
```

Expected output:

```json
{
  "service": "taskbridge-server",
  "status": "ok"
}
```

This confirms the server is running correctly.

---

# Step 4: Create HTTP Check Job

```powershell
$json = Get-Content .\examples\create-http-check-job.json -Raw

Invoke-RestMethod `
-Uri "http://localhost:8080/jobs" `
-Method POST `
-ContentType "application/json" `
-Body $json
```

Purpose:

* Creates a HTTP monitoring job
* Adds it to the scheduler queue

Expected output:

```text
status : PENDING
```

After the agent executes it:

```text
status : SUCCESS
```

---

# Step 5: Create TCP Check Job

```powershell
$json = Get-Content .\examples\create-tcp-check-job.json -Raw

Invoke-RestMethod `
-Uri "http://localhost:8080/jobs" `
-Method POST `
-ContentType "application/json" `
-Body $json
```

Purpose:

* Verifies TCP connectivity

---

# Step 6: Create File Exists Job

```powershell
$json = Get-Content .\examples\create-file-exists-job.json -Raw

Invoke-RestMethod `
-Uri "http://localhost:8080/jobs" `
-Method POST `
-ContentType "application/json" `
-Body $json
```

Purpose:

* Checks whether a file exists

---

# Step 7: Create Checksum Job

```powershell
$json = Get-Content .\examples\create-checksum-job.json -Raw

Invoke-RestMethod `
-Uri "http://localhost:8080/jobs" `
-Method POST `
-ContentType "application/json" `
-Body $json
```

Purpose:

* Computes SHA256 checksum

---

# Step 8: Create Write File Job

```powershell
$json = Get-Content .\examples\create-write-file-job.json -Raw

Invoke-RestMethod `
-Uri "http://localhost:8080/jobs" `
-Method POST `
-ContentType "application/json" `
-Body $json
```

Purpose:

* Writes content to a file

---

# Step 9: Create Copy File Job

```powershell
$json = Get-Content .\examples\create-copy-file-job.json -Raw

Invoke-RestMethod `
-Uri "http://localhost:8080/jobs" `
-Method POST `
-ContentType "application/json" `
-Body $json
```

Purpose:

* Copies a file from source to destination

---

# Step 10: Create Wait Job

```powershell
$json = Get-Content .\examples\create-wait-job.json -Raw

Invoke-RestMethod `
-Uri "http://localhost:8080/jobs" `
-Method POST `
-ContentType "application/json" `
-Body $json
```

Purpose:

* Simulates long-running execution

---

# Step 11: List All Jobs

```powershell
Invoke-RestMethod http://localhost:8080/jobs
```

Expected output:

```text
SUCCESS
RUNNING
PENDING
```

depending on current execution state.

This demonstrates:

* Job creation
* Scheduling
* Execution
* Lifecycle tracking

---

# Step 12: List Registered Agents

```powershell
Invoke-RestMethod http://localhost:8080/agents
```

Expected output:

```text
online : true
```

This verifies:

* Agent registration
* Heartbeats
* Online status tracking

---

# Step 13: Run Tests

```powershell
go test ./... -v
```

Run race detection:

```powershell
go test ./... -race
```

Generate coverage:

```powershell
go test ./... -coverprofile=coverage.out
go tool cover -func=coverage.out
```

---

# Step 14: Build Binaries

```powershell
go build -o bin/taskbridge-server ./cmd/server

go build -o bin/taskbridge-agent ./cmd/agent
```

---

# Demo Video Flow (1–2 Minutes)

1. Start Server
2. Start Agent
3. Verify Health Endpoint
4. Create HTTP Job
5. Show Job Success
6. Create Multiple Job Types
7. Show Agent Polling & Heartbeats
8. List Jobs
9. List Agents
10. End Demo

This sequence demonstrates all major TaskBridge requirements and stretch goals.
