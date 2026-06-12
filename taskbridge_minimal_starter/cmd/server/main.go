package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"taskbridge/internal/api"
	"taskbridge/internal/store"
)

func main() {
	// -----------------------------------------------------------------------
	// Configuration
	// -----------------------------------------------------------------------
	addr := flag.String("addr", ":8080", "server listen address")
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
	// Dependencies
	// -----------------------------------------------------------------------
	memStore := store.NewMemoryStore()

	srv := api.NewServer(memStore, logger, *authToken)

	// Start background agent health checker:
	// Check every 15 seconds, mark offline if no heartbeat for 30 seconds.
	srv.StartAgentHealthChecker(15*time.Second, 30*time.Second)

	// -----------------------------------------------------------------------
	// HTTP Server with graceful shutdown
	// -----------------------------------------------------------------------
	httpServer := &http.Server{
		Addr:         *addr,
		Handler:      srv.Handler(),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Channel to listen for OS signals.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		logger.Info("TaskBridge server starting",
			"addr", *addr,
			"auth", *authToken != "",
		)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	// Block until we receive a signal.
	sig := <-quit
	logger.Info("shutting down server", "signal", sig.String())

	// Give active requests up to 10 seconds to finish.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		logger.Error("forced shutdown", "error", err)
		os.Exit(1)
	}

	fmt.Println("server stopped gracefully")
}
