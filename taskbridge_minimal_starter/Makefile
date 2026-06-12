.PHONY: build-server build-agent run-server run-agent test test-race test-cover vet clean

# Build
build-server:
	go build -o bin/taskbridge-server ./cmd/server

build-agent:
	go build -o bin/taskbridge-agent ./cmd/agent

build: build-server build-agent

# Run
run-server:
	go run ./cmd/server --addr :8080

run-agent:
	go run ./cmd/agent --server http://localhost:8080 --id agent-dev-1

# Test
test:
	go test ./... -v -count=1

test-race:
	go test ./... -v -race -count=1

test-cover:
	go test ./... -v -race -coverprofile=coverage.out -count=1
	go tool cover -func=coverage.out

# Quality
vet:
	go vet ./...

# Clean
clean:
	rm -rf bin/ coverage.out
