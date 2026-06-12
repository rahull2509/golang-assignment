package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"taskbridge/internal/agent"
	"taskbridge/internal/executor"
)

func main() {
	// -----------------------------------------------------------------------
	// Configuration
	// -----------------------------------------------------------------------
	serverURL := flag.String("server", "http://localhost:8080", "TaskBridge server URL")
	agentID := flag.String("id", "agent-dev-1", "agent identifier")
	capabilities := flag.String("capabilities", "http_check,tcp_check,file_exists,checksum,copy_file,write_file,wait", "comma-separated job capabilities")
	pollInterval := flag.Duration("poll-interval", 3*time.Second, "job polling interval")
	authToken := flag.String("auth-token", "", "shared auth token (empty = disabled)")
	logJSON := flag.Bool("log-json", false, "output logs in JSON format")
	flag.Parse()

	// -----------------------------------------------------------------------
	// Logger
	// -----------------------------------------------------------------------
	var handler slog.Handler
	if *logJSON {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	} else {
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	}
	logger := slog.New(handler)
	slog.SetDefault(logger)

	// -----------------------------------------------------------------------
	// Executor registry
	// -----------------------------------------------------------------------
	registry := executor.NewRegistry()
	registry.Register(&executor.HTTPCheckExecutor{})
	registry.Register(&executor.TCPCheckExecutor{})
	registry.Register(&executor.FileExistsExecutor{})
	registry.Register(&executor.ChecksumExecutor{})
	registry.Register(&executor.CopyFileExecutor{})
	registry.Register(&executor.WriteFileExecutor{})
	registry.Register(&executor.WaitExecutor{})

	// -----------------------------------------------------------------------
	// Agent client + worker
	// -----------------------------------------------------------------------
	client := agent.NewClient(*serverURL, *agentID, *authToken)
	caps := strings.Split(*capabilities, ",")

	worker := agent.NewWorker(client, registry, logger, caps, *pollInterval)

	// -----------------------------------------------------------------------
	// Graceful shutdown
	// -----------------------------------------------------------------------
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-quit
		logger.Info("received shutdown signal", "signal", sig.String())
		cancel()
	}()

	logger.Info("TaskBridge agent starting",
		"server", *serverURL,
		"agent_id", *agentID,
		"capabilities", caps,
		"poll_interval", *pollInterval,
	)

	if err := worker.Run(ctx); err != nil {
		logger.Error("agent failed", "error", err)
		os.Exit(1)
	}
}
